package shopping

type CartCreated struct {
	CartID string // Unique identifier for the shopping cart
}

type CartItemAdded struct {
	Product Product // Product being added to the cart
}

type CartItemRemoved struct {
	ProductID string  // ID of the product being removed from the cart
	Product   Product // Product being removed from the cart
}

type CartDiscountApplied struct {
	Discount float64 // Discount amount applied to the cart
}

type UserCreated struct {
	ID    string // Unique identifier for the user
	Name  string // Name of the user
	Email string // Email of the user
}

type CartAddedToUser struct {
	CartID string `json:"cart_id"` // ID of the cart added to the user
}

type ProductCreated struct {
	ID    string  // Unique identifier for the product
	Name  string  // Name of the product
	Price float64 // Price of the product
}

type ProductNameUpdated struct {
	Name string // Name of the product
}

type ProductPriceUpdated struct {
	Price float64 // Price of the product
}

type ProductDeleted struct {
	ID string // Unique identifier for the product
}
