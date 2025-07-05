package collection

import "github.com/blinkinglight/gobeego/apps/shopping"

type Product struct {
	Products []shopping.Product `json:"products"` // List of products in the collection
}
