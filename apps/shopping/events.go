package shopping

type CartCreated struct {
	CartID string // Unique identifier for the shopping cart
}

type CartItemAdded struct {
	Product Product // Product being added to the cart
}

type CartItemRemoved struct {
	ProductID string // ID of the product being removed from the cart
}

type CartDiscountApplied struct {
	Discount float64 // Discount amount applied to the cart
}

type Product struct {
	ID    string  // Unique identifier for the product
	Name  string  // Name of the product
	Price float64 // Price of the product
}

type UserCreated struct {
	ID    string // Unique identifier for the user
	Name  string // Name of the user
	Email string // Email of the user
}

type CartAddedToUser struct {
	CartID string `json:"cart_id"` // ID of the cart added to the user
}
