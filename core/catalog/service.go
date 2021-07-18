package catalog

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/smfg-catalog/core"
)

func NewService(repo Repository, q Queue, productExchange string) *service {
	return &service{repo: repo, queue: q, productExchange: productExchange}
}

type Service interface {
	GetProduct(ctx context.Context, sku string) (Product, error)
	CreateProduct(ctx context.Context, product Product) error
}

type service struct {
	repo            Repository
	queue           Queue
	productExchange string
}

func (s *service) CreateProduct(ctx context.Context, product Product) error {
	const funcName = "CreateProduct"

	dbProduct, err := s.repo.GetProduct(ctx, product.Sku)
	if err != nil && !errors.Is(err, core.ErrNotFound) {
		return errors.WithStack(err)
	}

	if dbProduct.Sku != "" {
		log.Debug().
			Str("func", funcName).
			Str("sku", dbProduct.Sku).
			Msg("product already exists")
		return nil
	}

	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Info().
		Str("func", funcName).
		Str("sku", product.Sku).
		Str("upc", product.Upc).
		Msg("creating product")

	if err = s.repo.SaveProduct(ctx, product, tx); err != nil {
		rollback(ctx, tx, err)
		return errors.WithStack(err)
	}

	if err = tx.Commit(ctx); err != nil {
		rollback(ctx, tx, err)
		return errors.WithStack(err)
	}

	return nil
}

func (s *service) GetProduct(ctx context.Context, sku string) (Product, error) {
	const funcName = "GetProduct"

	log.Info().
		Str("func", funcName).
		Str("sku", sku).
		Msg("getting product")

	product, err := s.repo.GetProduct(ctx, sku)
	if err != nil {
		return product, errors.WithStack(err)
	}
	return product, nil
}

func rollback(ctx context.Context, tx core.Transaction, err error) {
	e := tx.Rollback(ctx)
	if e != nil {
		log.Warn().Err(err).Msg("failed to rollback")
	}
}

type Repository interface {
	SaveProduct(ctx context.Context, product Product, tx ...core.Transaction) error
	GetProduct(ctx context.Context, sku string, tx ...core.Transaction) (Product, error)
	BeginTransaction(ctx context.Context) (core.Transaction, error)
}

type Queue interface {
	PublishProduct(ctx context.Context, product Product) error
}
