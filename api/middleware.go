package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/rs/zerolog/log"
)

const DefaultPageLimit = 50

type CtxKey string

const (
	CtxKeyLimit  CtxKey = "limit"
	CtxKeyOffset CtxKey = "offset"
	CtxKeyUser   CtxKey = "user"
)

var (
	urlHitCount *prometheus.CounterVec
	urlLatency  *prometheus.SummaryVec
)

func Paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		var err error
		limit := DefaultPageLimit
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				limit = DefaultPageLimit
			}
		}

		offset := 0
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				offset = 0
			}
		}

		log.Debug().Int("limit", limit).Int("offset", offset).Send()
		ctx := context.WithValue(r.Context(), CtxKeyLimit, limit)
		ctx = context.WithValue(ctx, CtxKeyOffset, offset)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Logging(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			log.Trace().
				Str("method", r.Method).
				Str("host", r.Host).
				Str("uri", r.RequestURI).
				Str("proto", r.Proto).
				Int("status", ww.Status()).
				Int("bytes", ww.BytesWritten()).
				Dur("duration", time.Since(start)).Send()
		}()
		next.ServeHTTP(ww, r)
	}

	return http.HandlerFunc(fn)
}

func ConfigureMetrics() {
	urlHitCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_hit_count",
			Help: "Number of times the given url was hit",
		},
		[]string{"method", "url"},
	)
	urlLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "url_latency",
			Help:       "The latency quantiles for the given URL",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"method", "url"},
	)

	prometheus.MustRegister(urlHitCount)
	prometheus.MustRegister(urlLatency)
}

func Metrics(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		defer func() {
			ctx := chi.RouteContext(r.Context())

			if len(ctx.RoutePatterns) > 0 {
				dur := float64(time.Since(start).Milliseconds())
				urlLatency.WithLabelValues(ctx.RouteMethod, ctx.RoutePatterns[0]).Observe(dur)
				urlHitCount.WithLabelValues(ctx.RouteMethod, ctx.RoutePatterns[0]).Inc()
			}
		}()

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
