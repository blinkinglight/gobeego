package shopping

type Product struct {
	ID     string  `json:"id"`      // Unique identifier for the product
	CartID string  `json:"cart_id"` // ID of the cart this product belongs to
	Name   string  `json:"name"`    // Name of the product
	Price  float64 `json:"price"`   // Price of the product
}

type Cart struct {
	ID       string    // Unique identifier for the shopping cart
	Products []Product // List of products in the cart
	Total    float64   // Total price of the cart
	Discount float64   // Discount applied to the cart
}
