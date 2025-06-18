package banking

type DebitAccountCommand struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Ref           string `json:"ref"`
}

type CreditAccountCommand struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Ref           string `json:"ref"`
}
