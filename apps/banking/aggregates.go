package banking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
)

type AccountAggregate struct {
	ID       string
	Balance  int64 // pvz. centais
	Currency string
}

func (a *AccountAggregate) ApplyEvent(e *gen.EventEnvelope) error {
	if a.ID != e.AggregateId {
		log.Printf("shit")
		return fmt.Errorf("event does not belong to this account aggregate: %s != %s", a.ID, e.AggregateId)
	}
	ev, err := bee.UnmarshalEvent(e)
	if err != nil {
		return err
	}

	switch evt := ev.(type) {
	case *AccountDebited:
		a.Balance -= evt.Amount
	case *AccountCredited:
		a.Balance += evt.Amount
	default:
		return fmt.Errorf("unknown event type: %T", ev)
	}
	return nil
}

type PaymentAggregate struct {
	ID       string
	Balance  int64 // pvz. centais
	Currency string

	found bool
}

func (a *PaymentAggregate) ApplyEvent(e *gen.EventEnvelope) error {
	ev, err := bee.UnmarshalEvent(e)
	if err != nil {
		return err
	}
	a.found = true
	switch ev := ev.(type) {
	case *AccountCreated:
		if a.ID != ev.AccountID {
			return errors.New("event does not belong to this payment aggregate")
		}
		a.ID = ev.AccountID
		a.Balance = ev.Balance
		a.Currency = ev.Currency
	case *AccountDebited:
		if a.ID != ev.AccountID {
			return errors.New("event does not belong to this payment aggregate")
		}
		a.Balance -= ev.Amount
	case *AccountCredited:
		if a.ID != ev.AccountID {
			return errors.New("event does not belong to this payment aggregate")
		}
		a.Balance += ev.Amount
	default:
		return errors.New("unknown event type")
	}

	return nil
}

func (a *PaymentAggregate) ApplyCommand(ctx context.Context, c *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {

	cmd, err := bee.UnmarshalCommand(c)
	if err != nil {
		return nil, err
	}

	switch cmd := cmd.(type) {
	case *CreateAccountCommand:
		if cmd.Balance < 0 {
			return nil, errors.New("initial balance cannot be negative")
		}
		if cmd.Currency == "" {
			return nil, errors.New("currency cannot be empty")
		}
		a.ID = cmd.AccountID
		a.Balance = cmd.Balance
		a.Currency = cmd.Currency
		ev := &AccountCreated{
			AccountID: cmd.AccountID,
			Currency:  cmd.Currency,
			Balance:   cmd.Balance,
			Ref:       cmd.Ref,
			Timestamp: c.Timestamp.AsTime().Unix(),
		}
		var event *gen.EventEnvelope = &gen.EventEnvelope{AggregateId: cmd.AccountID}
		event.AggregateType = "payments"
		event.EventType = "credited"
		b, _ := json.Marshal(ev)
		event.Payload = b
		return []*gen.EventEnvelope{event}, nil

	case *DebitAccountCommand:
		if cmd.Amount <= 0 {
			return nil, errors.New("amount must be greater than zero")
		}
		if a.Balance < cmd.Amount {
			return nil, errors.New("insufficient funds")
		}
		a.Balance -= cmd.Amount
		ev := &AccountDebited{
			AccountID:  cmd.FromAccountID,
			Amount:     cmd.Amount,
			Ref:        cmd.Ref,
			NewBalance: a.Balance,
		}
		var event *gen.EventEnvelope = &gen.EventEnvelope{AggregateId: cmd.FromAccountID}
		event.AggregateType = "payments"
		event.EventType = "debited"
		b, _ := json.Marshal(ev)
		event.Payload = b

		var eventc *gen.EventEnvelope = &gen.EventEnvelope{AggregateId: cmd.ToAccountID}
		eventc.AggregateType = "payments"
		evc := &AccountCredited{
			AccountID: cmd.ToAccountID,
			Amount:    cmd.Amount,
			Ref:       cmd.Ref,
		}
		eventc.EventType = "credited"
		b2, _ := json.Marshal(evc)
		eventc.Payload = b2
		return []*gen.EventEnvelope{event, eventc}, nil
	case *CreditAccountCommand:
		if cmd.Amount <= 0 {
			return nil, errors.New("amount must be greater than zero")
		}
		a.Balance += cmd.Amount
		ev := &AccountCredited{
			AccountID:  cmd.ToAccountID,
			Amount:     cmd.Amount,
			Ref:        cmd.Ref,
			NewBalance: a.Balance,
		}
		var event *gen.EventEnvelope = &gen.EventEnvelope{AggregateId: cmd.ToAccountID}
		event.AggregateType = "payments"
		event.EventType = "credited"
		b, _ := json.Marshal(ev)
		event.Payload = b
		return []*gen.EventEnvelope{event}, nil
	default:
		return nil, fmt.Errorf("unknown command type: %T", cmd)
	}

}
