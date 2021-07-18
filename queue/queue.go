package queue

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sksmith/bunnyq"
	"github.com/sksmith/smfg-catalog/core/catalog"
)

type productQueue struct {
	queue           *bunnyq.BunnyQ
	productExchange string
}

func New(bq *bunnyq.BunnyQ, productExchange string) *productQueue {
	return &productQueue{queue: bq, productExchange: productExchange}
}

func (p *productQueue) PublishProduct(ctx context.Context, product catalog.Product) error {
	body, err := json.Marshal(product)
	if err != nil {
		return errors.WithMessage(err, "failed to serialize message for queue")
	}
	if err = p.queue.Publish(ctx, p.productExchange, body); err != nil {
		return errors.WithMessage(err, "failed to send inventory update to queue")
	}
	return nil
}
