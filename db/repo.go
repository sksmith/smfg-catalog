package db

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"github.com/sksmith/smfg-catalog/core"
	"github.com/sksmith/smfg-catalog/core/catalog"
)

type dbRepo struct {
	conn core.Conn
}

func NewPostgresRepo(conn core.Conn) *dbRepo {
	return &dbRepo{
		conn: conn,
	}
}

func (d *dbRepo) SaveProduct(ctx context.Context, product catalog.Product, txs ...core.Transaction) error {
	m := StartMetric("SaveProduct")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}
	ct, err := tx.Exec(ctx, `
		UPDATE products
           SET upc = $2, name = $3
         WHERE sku = $1;`,
		product.Sku, product.Upc, product.Name)
	if err != nil {
		m.Complete(nil)
		return errors.WithStack(err)
	}
	if ct.RowsAffected() == 0 {
		_, err := tx.Exec(ctx, `
		INSERT INTO products (sku, upc, name)
                      VALUES ($1, $2, $3);`,
			product.Sku, product.Upc, product.Name)
		if err != nil {
			m.Complete(err)
			return err
		}
	}
	m.Complete(nil)
	return nil
}

func (d *dbRepo) GetProduct(ctx context.Context, sku string, txs ...core.Transaction) (catalog.Product, error) {
	m := StartMetric("GetProduct")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}

	product := catalog.Product{}
	err := tx.QueryRow(ctx, `SELECT sku, upc, name FROM products WHERE sku = $1`, sku).
		Scan(&product.Sku, &product.Upc, &product.Name)

	if err != nil {
		m.Complete(err)
		if err == pgx.ErrNoRows {
			return product, errors.WithStack(core.ErrNotFound)
		}
		return product, errors.WithStack(err)
	}

	m.Complete(nil)
	return product, nil
}

func (d *dbRepo) BeginTransaction(ctx context.Context) (core.Transaction, error) {
	tx, err := d.conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
