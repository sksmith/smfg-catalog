package queue

import (
	"context"

	"github.com/sksmith/smfg-catalog/core/catalog"
)

type MockQueue struct {
	PublishProductFunc func(ctx context.Context, product catalog.Product) error
}

func NewMockQueue() *MockQueue {
	return &MockQueue{
		PublishProductFunc: func(ctx context.Context, product catalog.Product) error {
			return nil
		},
	}
}

func (m *MockQueue) PublishProduct(ctx context.Context, product catalog.Product) error {
	return m.PublishProductFunc(ctx, product)
}
