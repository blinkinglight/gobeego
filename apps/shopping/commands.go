package shopping

type CartCreate struct {
	CartID string // Unique identifier for the shopping cart
}

type CartItemAdd struct {
	Product Product // Product being added to the cart
}

type CartItemRemove struct {
	ProductID string // ID of the product being removed from the cart
}

type CartDiscountApply struct {
	Discount float64 // Discount amount applied to the cart
}

type UserCreate struct {
	ID    string // Unique identifier for the user
	Name  string // Name of the user
	Email string // Email of the user
}

type CartAddToUser struct {
	CartID string // ID of the cart added to the user
}
