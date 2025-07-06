package main

import (
	"context"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/gobeego/apps/shopping"
	"github.com/blinkinglight/gobeego/pkg/appctx"
	"github.com/blinkinglight/gobeego/pkg/rwdb"
	"github.com/blinkinglight/gobeego/pkg/utils"
)

type ProductService struct {
	Ctx context.Context
}

func (s *ProductService) Handle(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	event, err := bee.UnmarshalCommand(m)
	if err != nil {
		return nil, err
	}
	// db := appctx.DB(s.Ctx)
	switch event := event.(type) {
	case *shopping.ProductCreate:
		return []*gen.EventEnvelope{{
			AggregateId: m.AggregateId,
			EventType:   "created",
			Payload:     utils.MustMarshal(&shopping.ProductCreated{Name: event.Name, Price: event.Price}),
		}}, nil
	case *shopping.ProductUpdateName:
		return []*gen.EventEnvelope{{
			AggregateId: m.AggregateId,
			EventType:   "name_updated",
			Payload:     utils.MustMarshal(&shopping.ProductNameUpdated{Name: event.Name}),
		}}, nil
	case *shopping.ProductUpdatePrice:
		return []*gen.EventEnvelope{{
			AggregateId: m.AggregateId,
			EventType:   "price_updated",
			Payload:     utils.MustMarshal(&shopping.ProductPriceUpdated{Price: event.Price}),
		}}, nil
	case *shopping.ProductDelete:
		return []*gen.EventEnvelope{{
			AggregateId: m.AggregateId,
			EventType:   "deleted",
			Payload:     utils.MustMarshal(&shopping.ProductDeleted{}),
		}}, nil
	default:
		return nil, nil // Ignore other command types
	}
}

type ProductProjection struct {
	Ctx context.Context
}

func (p *ProductProjection) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return err
	}
	db := appctx.DB(p.Ctx)
	switch event := event.(type) {
	case *shopping.ProductCreated:
		return db.WriteTX(p.Ctx, func(tx *rwdb.Tx) error {
			product := &shopping.Product{
				ID:    e.AggregateId,
				Name:  event.Name,
				Price: event.Price,
			}
			return tx.Create(product).Error
		})
	case *shopping.ProductNameUpdated:
		return db.WriteTX(p.Ctx, func(tx *rwdb.Tx) error {
			product := &shopping.Product{
				Name: event.Name,
			}
			return tx.Model(&shopping.Product{}).Select("name").Where("id = ?", e.AggregateId).Updates(product).Error
		})
	case *shopping.ProductPriceUpdated:
		return db.WriteTX(p.Ctx, func(tx *rwdb.Tx) error {
			product := &shopping.Product{
				Price: event.Price,
			}
			return tx.Model(&shopping.Product{}).Select("price").Where("id = ?", e.AggregateId).Updates(product).Error
		})
	case *shopping.ProductDeleted:
		return db.WriteTX(p.Ctx, func(tx *rwdb.Tx) error {
			return tx.Where("id = ?", e.AggregateId).Delete(&shopping.Product{}).Error
		})
	default:
		return nil // Ignore other event types
	}
	return nil
}
