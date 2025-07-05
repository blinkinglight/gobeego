package collection

import "github.com/blinkinglight/gobeego/apps/shopping"

type Cart struct {
	Count    int                `json:"count"`    // Number of products in the cart
	CartID   string             `json:"cart_id"`  // Unique identifier for the shopping cart
	Products []shopping.Product `json:"products"` // List of products in the cart
	Total    float64            `json:"total"`    // Total price of the cart
	Discount float64            `json:"discount"` // Discount applied to the cart
}
