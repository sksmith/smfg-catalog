package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/bunnyq"
	"github.com/sksmith/smfg-catalog/api"
	"github.com/sksmith/smfg-catalog/core/catalog"
	"github.com/sksmith/smfg-catalog/db"
	"github.com/sksmith/smfg-catalog/queue"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	ApplicationName = "smfg-catalog"
	Revision        = "1"
)

var (
	AppVersion  string
	Sha1Version string
	BuildTime   string

	configUrl    = os.Getenv("CONFIG_SERVER_URL")
	configBranch = os.Getenv("CONFIG_SERVER_BRANCH")
	configUser   = os.Getenv("CONFIG_SERVER_USER")
	configPass   = os.Getenv("CONFIG_SERVER_PASS")
	profile      = os.Getenv("PROFILE")
	printConfigs = getPrintConfigs()
)

func getPrintConfigs() bool {
	v, err := strconv.ParseBool(os.Getenv("PRINT_CONFIGS"))
	if err != nil {
		return false
	}
	return v
}

func main() {
	ctx := context.Background()

	config := loadConfigs()

	configLogging(config)
	printLogHeader(config)
	dbPool := configDatabase(ctx, config)
	bq := rabbit(config)
	q := configInventoryQueue(bq, config)

	log.Info().Msg("creating catalog service...")
	ir := db.NewPostgresRepo(dbPool)
	catalogService := catalog.NewService(ir, q, config.QProductExchange)

	log.Info().Msg("configuring metrics...")
	api.ConfigureMetrics()

	log.Info().Msg("configuring router...")
	r := configureRouter(catalogService)

	log.Info().Str("port", config.Port).Msg("listening")
	log.Fatal().Err(http.ListenAndServe(":"+config.Port, r))
}

func configInventoryQueue(bq *bunnyq.BunnyQ, config *Config) (q catalog.Queue) {
	if config.QMock {
		log.Info().Msg("creating mock queue...")
		return queue.NewMockQueue()
	} else {
		log.Info().Msg("connecting to rabbitmq...")
		return queue.New(bq, config.QProductExchange)
	}
}

func loadConfigs() (config *Config) {
	var err error

	if profile == "local" || profile == "" {
		log.Info().Msg("loading local configurations...")
		config, err = LoadLocalConfigs()
	} else {
		config, err = LoadRemoteConfigs(configUrl, configBranch, configUser, configPass, profile)
	}
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configurations")
	}

	config.Revision = Revision

	return config
}

func rabbit(config *Config) *bunnyq.BunnyQ {
	osChannel := make(chan os.Signal, 1)
	signal.Notify(osChannel, syscall.SIGTERM)
	var bq *bunnyq.BunnyQ

	for {
		bq = bunnyq.New(context.Background(),
			bunnyq.Address{
				User: config.QUser,
				Pass: config.QPass,
				Host: config.QHost,
				Port: config.QPort,
			},
			osChannel,
			bunnyq.LogHandler(logger{}),
		)

		break
	}

	return bq
}

type logger struct {
}

func (l logger) Log(_ context.Context, level bunnyq.LogLevel, msg string, data map[string]interface{}) {
	var evt *zerolog.Event
	switch level {
	case bunnyq.LogLevelTrace:
		evt = log.Trace()
	case bunnyq.LogLevelDebug:
		evt = log.Debug()
	case bunnyq.LogLevelInfo:
		evt = log.Info()
	case bunnyq.LogLevelWarn:
		evt = log.Warn()
	case bunnyq.LogLevelError:
		evt = log.Error()
	case bunnyq.LogLevelNone:
		evt = log.Info()
	default:
		evt = log.Info()
	}

	for k, v := range data {
		evt.Interface(k, v)
	}

	evt.Msg(msg)
}

func printLogHeader(c *Config) {
	if c.LogText {
		log.Info().Msg("=============================================")
		log.Info().Msg(fmt.Sprintf("    Application: %s", ApplicationName))
		log.Info().Msg(fmt.Sprintf("       Revision: %s", c.Revision))
		log.Info().Msg(fmt.Sprintf("        Profile: %s", profile))
		log.Info().Msg(fmt.Sprintf("  Config Server: %s - %s", configUrl, configBranch))
		log.Info().Msg(fmt.Sprintf("    Tag Version: %s", AppVersion))
		log.Info().Msg(fmt.Sprintf("   Sha1 Version: %s", Sha1Version))
		log.Info().Msg(fmt.Sprintf("     Build Time: %s", BuildTime))
		log.Info().Msg("=============================================")
	} else {
		log.Info().Str("application", ApplicationName).
			Str("revision", c.Revision).
			Str("version", AppVersion).
			Str("sha1ver", Sha1Version).
			Str("build-time", BuildTime).
			Str("profile", profile).
			Str("config-url", configUrl).
			Str("config-branch", configBranch).
			Send()
	}
}

func configDatabase(ctx context.Context, config *Config) (dbPool *pgxpool.Pool) {
	log.Info().Str("host", config.DbHost).Str("name", config.DbName).Msg("connecting to the database...")
	var err error

	if config.DbMigrate {
		log.Info().Msg("executing migrations")

		if err = db.RunMigrations(
			config.DbHost,
			config.DbName,
			config.DbPort,
			config.DbUser,
			config.DbPass,
			config.DbClean); err != nil {
			log.Warn().Err(err).Msg("error executing migrations")
		}
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
		config.DbHost, config.DbPort, config.DbUser, config.DbPass, config.DbName)

	for {
		dbPool, err = db.ConnectDb(ctx, connStr, db.MinPoolConns(10), db.MaxPoolConns(50))
		if err != nil {
			log.Error().Err(err).Msg("failed to create connection pool... retrying")
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	return dbPool
}

func configureRouter(service catalog.Service) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(api.Metrics)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(api.Logging)

	r.Handle("/metrics", promhttp.Handler())
	r.Route("/api", func(r chi.Router) {
		r.Route("/product", catalogApi(service))
	})

	return r
}

func catalogApi(s catalog.Service) func(r chi.Router) {
	catApi := api.NewCatalogApi(s)
	return catApi.ConfigureRouter
}

func configLogging(config *Config) {
	log.Info().Msg("configuring logging...")

	if config.LogText {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	level, err := zerolog.ParseLevel(config.LogLevel)
	if err != nil {
		log.Warn().Str("loglevel", config.LogLevel).Err(err).Msg("defaulting to info")
		level = zerolog.InfoLevel
	}
	log.Info().Str("loglevel", level.String()).Msg("setting log level")
	zerolog.SetGlobalLevel(level)
}
