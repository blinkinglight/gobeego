package test

import (
	"encoding/json"

	"github.com/blinkinglight/bee/gen"
)

type CreateAccountCommand struct {
	AccountId string `json:"account_id"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
	Ref       string `json:"ref"`
}

type DebitAccountCommand struct {
	FromAccountId string `json:"from_account_id"`
	ToAccountId   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Ref           string `json:"ref"`
}

type CreditAccountCommand struct {
	FromAccountId string `json:"from_account_id"`
	ToAccountId   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Ref           string `json:"ref"`
}

func CreateAccount(AccountId string, Currency string, Balance int64, Ref string) *gen.CommandEnvelope {
	payload := &CreateAccountCommand{
		AccountId: AccountId,
		Currency:  Currency,
		Balance:   Balance,
		Ref:       Ref,
	}
	b, _ := json.Marshal(payload)

	cmd := &gen.CommandEnvelope{
		AggregateId: AccountId,
		Aggregate:   "accounts",
		CommandType: "create",
		Payload:     b,
	}
	return cmd
}

func DebitAccount(FromAccountId string, ToAccountId string, Amount int64, Ref string) *gen.CommandEnvelope {
	payload := &DebitAccountCommand{
		FromAccountId: FromAccountId,
		ToAccountId:   ToAccountId,
		Amount:        Amount,
		Ref:           Ref,
	}
	b, _ := json.Marshal(payload)

	cmd := &gen.CommandEnvelope{
		AggregateId: FromAccountId,
		Aggregate:   "accounts",
		CommandType: "debit",
		Payload:     b,
	}
	return cmd
}

func CreditAccount(FromAccountId string, ToAccountId string, Amount int64, Ref string) *gen.CommandEnvelope {
	payload := &CreditAccountCommand{
		FromAccountId: FromAccountId,
		ToAccountId:   ToAccountId,
		Amount:        Amount,
		Ref:           Ref,
	}
	b, _ := json.Marshal(payload)

	cmd := &gen.CommandEnvelope{
		AggregateId: ToAccountId,
		Aggregate:   "accounts",
		CommandType: "credit",
		Payload:     b,
	}
	return cmd
}
