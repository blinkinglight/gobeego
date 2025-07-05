package banking

import (
	"context"
	"errors"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
)

type PaymentService struct {
	Ctx context.Context
}

func (s *PaymentService) Handle(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := &PaymentAggregate{ID: m.AggregateId}
	bee.Replay(s.Ctx, agg, ro.WithAggreate(m.Aggregate), ro.WithAggregateID(m.AggregateId))

	if agg.found && m.CommandType == CreateCommand {
		return nil, errors.New("account already exists")
	} else if !agg.found && m.CommandType != CreateCommand {
		return nil, errors.New("account does not exist")
	}

	return agg.ApplyCommand(s.Ctx, m)
}
