package main

import (
	"context"
	"errors"
	"fmt"

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

type ProductLiveView struct {
	err     error
	Product shopping.Product `json:"product"` // Current state of the product
}

func (p *ProductLiveView) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	p.err = nil                  // Reset error on each event
	p.Product.ID = e.AggregateId // Set the product ID from the event
	switch event := event.(type) {
	case *shopping.ProductCreated:
		// Handle product creation
		p.Product.Name = event.Name
		p.Product.Price = event.Price
		return nil // Ignore product creation event
	case *shopping.ProductNameUpdated:
		// Handle product name update
		p.Product.Name = event.Name
	case *shopping.ProductPriceUpdated:
		// Handle product price update
		p.Product.Price = event.Price
	case *shopping.ProductDeleted:
		// Handle product deletion
	default:
		p.err = errors.New("unsupported event type for UpdateProductLiveProjection")
	}
	return nil
}

type UpdateProductLiveProjection struct {
	err      error
	Products []shopping.Product `json:"products"` // List of products in the projection
}

func (p *UpdateProductLiveProjection) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	p.err = nil // Reset error on each event
	switch event := event.(type) {
	case *shopping.ProductCreated:
		p.Products = append(p.Products, shopping.Product{
			ID:    e.AggregateId,
			Name:  event.Name,
			Price: event.Price,
		})
		return nil // Ignore product creation event
	default:
		p.err = errors.New("unsupported event type for UpdateProductLiveProjection")
	}
	return nil
}

type CartCounterLiveProjection struct {
	Count int     `json:"count"` // Count of items in the cart
	Total float64 `json:"total"` // Total price of items in the cart
}

// ApplyEvent applies an event to the CartCounterLiveProjection
func (c *CartCounterLiveProjection) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	switch event := event.(type) {
	case *shopping.CartItemAdded:
		c.Count++
		c.Total += event.Product.Price
	case *shopping.CartItemRemoved:
		c.Count--
		c.Total -= event.Product.Price
	default:
		return nil // Ignore other event types
	}
	return nil
}

type CartProjection struct {
	Items []shopping.Product `json:"history"` // History of events applied to the aggregate
}

func (a *CartProjection) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	switch event := event.(type) {
	case *shopping.CartItemAdded:
		a.Items = append(a.Items, shopping.Product{
			ID:    event.Product.ID,
			Name:  event.Product.Name,
			Price: event.Product.Price,
		})
	case *shopping.CartItemRemoved:
		for i, p := range a.Items {
			if p.ID == event.ProductID {
				a.Items = append(a.Items[:i], a.Items[i+1:]...)
				break
			}
		}
	default:
		return nil // Ignore other event types
	}
	return nil
}
