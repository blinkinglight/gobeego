package shopping

import (
	"fmt"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/gobeego/pkg/utils"
)

type ShoppingCartAggregate struct {
	ID       string
	Items    []Product
	Total    float64
	Discount float64

	found bool
}

func (s *ShoppingCartAggregate) ApplyEvent(e *gen.EventEnvelope) error {
	ev, err := bee.UnmarshalEvent(e)
	if err != nil {
		return err
	}
	s.found = true
	switch evt := ev.(type) {
	case *CartItemAdded:
		s.Items = append(s.Items, evt.Product)
		s.Total += evt.Product.Price
	case *CartItemRemoved:
		for i, item := range s.Items {
			if item.ID == evt.ProductID {
				s.Total -= item.Price
				s.Items = append(s.Items[:i], s.Items[i+1:]...)
				break
			}
		}
	case *CartDiscountApplied:
		s.Discount = evt.Discount
	default:
		return fmt.Errorf("unknown event type: %T", ev)
	}
	return nil
}

func (s *ShoppingCartAggregate) ApplyCommand(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {

	if s.found && m.CommandType == "create" {
		return nil, fmt.Errorf("shopping cart already exists: %s", s.ID)
	} else if !s.found && m.CommandType != "create" {
		return nil, fmt.Errorf("shopping cart does not exist: %s", s.ID)
	}

	cmd, _ := bee.UnmarshalCommand(m)

	switch cmd.(type) {
	case *CartCreate:
		if s.found {
			return nil, fmt.Errorf("shopping cart already exists: %s", m.AggregateId)
		}
		s.ID = m.AggregateId
		s.Items = []Product{}
		s.Total = 0
		s.Discount = 0
		return []*gen.EventEnvelope{{
			AggregateId: m.AggregateId,
			EventType:   "created",
			Payload:     m.Payload,
			Metadata:    m.Metadata,
		}}, nil

	case *CartItemAdd:
		return []*gen.EventEnvelope{{
			AggregateId: m.AggregateId,
			EventType:   "item_added",
			Payload:     m.Payload,
			Metadata:    m.Metadata,
		}}, nil
	case *CartItemRemove:
		input := m.ExtraMetadata.AsMap()
		var product Product
		utils.FromStructpbMap(input["product"].(utils.M), &product)

		return []*gen.EventEnvelope{{
			AggregateId: m.AggregateId,
			EventType:   "item_removed",
			Payload: utils.MustMarshal(&CartItemRemoved{
				ProductID: product.ID,
				Product:   product,
			}),
			Metadata: m.Metadata,
		}}, nil
	case *CartDiscountApply:
		return []*gen.EventEnvelope{{
			AggregateId: m.AggregateId,
			EventType:   "discount_applied",
			Payload:     m.Payload,
			Metadata:    m.Metadata,
		}}, nil
	default:
		return nil, fmt.Errorf("unknown command type: %T %v", cmd, m.CommandType)
	}
}

type UserAggregate struct {
	ID    string
	Name  string
	Email string
	Carts []string // List of cart IDs associated with the user
	found bool
}

func (u *UserAggregate) ApplyEvent(e *gen.EventEnvelope) error {
	ev, err := bee.UnmarshalEvent(e)
	if err != nil {
		return err
	}
	u.found = true
	switch evt := ev.(type) {
	case *UserCreated:
		u.ID = e.AggregateId
		u.Name = evt.Name
		u.Email = evt.Email
	case *CartAddedToUser:
		u.Carts = append(u.Carts, evt.CartID)
	default:
		return fmt.Errorf("unknown event type: %T", ev)
	}
	return nil
}

func (u *UserAggregate) ApplyCommand(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	if u.found && m.CommandType == "create" {
		return nil, fmt.Errorf("user already exists: %s", u.ID)
	} else if !u.found && m.CommandType != "create" {
		return nil, fmt.Errorf("user does not exist: %s", u.ID)
	}

	cmd, _ := bee.UnmarshalCommand(m)

	switch evt := cmd.(type) {
	case *UserCreate:
		if u.found {
			return nil, fmt.Errorf("user already exists: %s", m.AggregateId)
		}
		u.ID = m.AggregateId
		u.Name = evt.Name
		u.Email = evt.Email
		return []*gen.EventEnvelope{{
			EventType: "created",
			Payload:   m.Payload,
			Metadata:  m.Metadata,
		}}, nil

	case *CartAddToUser:
		events := []*gen.EventEnvelope{{
			AggregateId: m.Metadata["cart_id"],
			EventType:   "cart_added",
			Payload:     m.Payload,
			Metadata:    m.Metadata,
		}}
		return events, nil

	default:
		return nil, fmt.Errorf("unknown command type: %T %v", cmd, m.CommandType)
	}
}

type ProductAggregate struct {
	ID    string
	Name  string
	Price float64
	found bool
}

func (p *ProductAggregate) ApplyEvent(e *gen.EventEnvelope) error {
	ev, err := bee.UnmarshalEvent(e)
	if err != nil {
		return err
	}
	p.found = true
	switch evt := ev.(type) {
	case *ProductCreated:
		p.ID = e.AggregateId
		p.Name = evt.Name
		p.Price = evt.Price
	case *ProductNameUpdated:
		p.Name = evt.Name
	case *ProductPriceUpdated:
		p.Price = evt.Price
	case *ProductDeleted:
		p.ID = ""
		p.Name = ""
		p.Price = 0
	default:
		return fmt.Errorf("unknown event type: %T", ev)
	}
	return nil
}
