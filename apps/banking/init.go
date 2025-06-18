package banking

import "github.com/blinkinglight/bee"

func init() {
	bee.RegisterEvent[AccountDebited]("accounts", "debited")
	bee.RegisterEvent[AccountCredited]("accounts", "credited")

	bee.RegisterCommand[DebitAccountCommand]("accounts", "debit")
}
