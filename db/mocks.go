package db

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/sksmith/smfg-catalog/core"
	"github.com/sksmith/smfg-catalog/core/catalog"
)

type MockRepo struct {
	SaveProductFunc      func(ctx context.Context, product catalog.Product, tx ...core.Transaction) error
	GetProductFunc       func(ctx context.Context, sku string, tx ...core.Transaction) (catalog.Product, error)
	BeginTransactionFunc func(ctx context.Context) (core.Transaction, error)
}

func (r MockRepo) SaveProduct(ctx context.Context, product catalog.Product, tx ...core.Transaction) error {
	return r.SaveProductFunc(ctx, product, tx...)
}

func (r MockRepo) GetProduct(ctx context.Context, sku string, tx ...core.Transaction) (catalog.Product, error) {
	return r.GetProductFunc(ctx, sku, tx...)
}

func (r MockRepo) BeginTransaction(ctx context.Context) (core.Transaction, error) {
	return r.BeginTransactionFunc(ctx)
}

func NewMockRepo() MockRepo {
	return MockRepo{
		SaveProductFunc: func(ctx context.Context, product catalog.Product, tx ...core.Transaction) error { return nil },
		GetProductFunc: func(ctx context.Context, sku string, tx ...core.Transaction) (catalog.Product, error) {
			return catalog.Product{}, nil
		},
		BeginTransactionFunc: func(ctx context.Context) (core.Transaction, error) {
			return MockTransaction{}, nil
		},
	}
}

type MockTransaction struct {
}

func (m MockTransaction) Commit(_ context.Context) error {
	return nil
}

func (m MockTransaction) Rollback(_ context.Context) error {
	return nil
}

func (m MockTransaction) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m MockTransaction) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row {
	return nil
}

func (m MockTransaction) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return nil, nil
}

func (m MockTransaction) Begin(_ context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (m MockTransaction) Conn() core.Conn {
	return nil
}
