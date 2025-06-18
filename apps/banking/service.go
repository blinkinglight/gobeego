package banking

import (
	"context"
	"errors"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
)

type PaymentService struct{}

func (s *PaymentService) Handle(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := &PaymentAggregate{ID: m.AggregateId}
	bee.Replay(ctx, agg, ro.WithAggreate("payments"), ro.WithAggregateID(m.AggregateId))

	if agg.found && m.CommandType == "create" {
		return nil, errors.New("account already exists")
	} else if !agg.found && m.CommandType != "create" {
		return nil, errors.New("account does not exist")
	}

	return agg.ApplyCommand(ctx, m)
}
