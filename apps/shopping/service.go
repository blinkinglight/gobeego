package shopping

import (
	"context"
	"fmt"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
)

type CartService struct {
	Ctx context.Context
}

func (s *CartService) Handle(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := &ShoppingCartAggregate{ID: m.AggregateId}
	bee.Replay(s.Ctx, agg, ro.WithAggreate(m.Aggregate), ro.WithAggregateID(m.AggregateId))
	return agg.ApplyCommand(m)
}

type UserService struct {
	Ctx context.Context
}

func (s *UserService) Handle(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := &UserAggregate{ID: m.AggregateId}
	bee.Replay(s.Ctx, agg, ro.WithAggreate(m.Aggregate), ro.WithAggregateID(m.AggregateId))

	if agg.found && m.CommandType == "create" {
		return nil, fmt.Errorf("aggregate already exists: %s", m.AggregateId)
	} else if !agg.found && m.CommandType != "create" {
		return nil, fmt.Errorf("aggregate does not exist: %s", m.AggregateId)
	}

	events := []*gen.EventEnvelope{}
	cmdEvents, err := agg.ApplyCommand(m)
	events = append(events, cmdEvents...)

	if m.CommandType == "create" {
		if agg.found {
			return nil, fmt.Errorf("user already exists: %s", m.AggregateId)
		}
		events = append(events, &gen.EventEnvelope{
			EventType: "cart_added",
			Payload:   []byte(`{"cart_id": "` + m.Metadata["cart_id"] + `"}`),
			Metadata:  m.Metadata,
		})
	}

	return events, err
}
