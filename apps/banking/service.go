package banking

import (
	"context"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
)

type PaymentService struct{}

func (s *PaymentService) Handle(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := &PaymentAggregate{ID: m.AggregateId}
	bee.Replay(ctx, agg, ro.WithAggreate("payments"), ro.WithAggregateID(m.AggregateId))

	return agg.ApplyCommand(ctx, m)
}
