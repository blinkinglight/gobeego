package shopping

import (
	"context"
	"fmt"
	"log"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
	"github.com/blinkinglight/gobeego/pkg/utils"
	"google.golang.org/protobuf/types/known/structpb"
)

type CartService struct {
	Ctx context.Context
}

func (s *CartService) Handle(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := &ShoppingCartAggregate{ID: m.AggregateId}
	bee.Replay(s.Ctx, agg, ro.WithAggreate(m.Aggregate), ro.WithAggregateID(m.AggregateId))

	evt, err := bee.UnmarshalCommand(m)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	switch evt := evt.(type) {
	case *CartItemRemove:
		productAgg := &ProductAggregate{ID: evt.ProductID}
		bee.Replay(s.Ctx, productAgg, ro.WithAggreate("product"), ro.WithAggregateID(evt.ProductID))

		m.ExtraMetadata, err = structpb.NewStruct(map[string]any{
			"product": utils.ToStructpbMap(&Product{
				ID:    productAgg.ID,
				Name:  productAgg.Name,
				Price: productAgg.Price,
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create extra metadata for product: %w", err)
		}
	default:
		log.Printf("Unhandled command type: %T", evt)
	}

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
