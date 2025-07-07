package shopping

import "github.com/blinkinglight/bee"

func init() {
	bee.RegisterEvent[CartCreated]("cart", "created")
	bee.RegisterEvent[CartItemAdded]("cart", "item_added")
	bee.RegisterEvent[CartItemRemoved]("cart", "item_removed")
	bee.RegisterEvent[CartDiscountApplied]("cart", "discount_applied")

	bee.RegisterCommand[CartCreate]("cart", "create")
	bee.RegisterCommand[CartItemAdd]("cart", "add_item")
	bee.RegisterCommand[CartItemRemove]("cart", "remove_item")
	bee.RegisterCommand[CartDiscountApply]("cart", "apply_discount")

	bee.RegisterEvent[UserCreated]("user", "created")
	bee.RegisterCommand[UserCreate]("user", "create")
	bee.RegisterCommand[CartAddToUser]("user", "add_cart")
	bee.RegisterEvent[CartAddedToUser]("user", "cart_added")

	bee.RegisterEvent[ProductCreated]("product", "created")
	bee.RegisterEvent[ProductNameUpdated]("product", "name_updated")
	bee.RegisterEvent[ProductPriceUpdated]("product", "price_updated")
	bee.RegisterEvent[ProductDeleted]("product", "deleted")

	bee.RegisterCommand[ProductCreate]("product", "create")
	bee.RegisterCommand[ProductUpdateName]("product", "update_name")
	bee.RegisterCommand[ProductUpdatePrice]("product", "update_price")
	bee.RegisterCommand[ProductDelete]("product", "delete")
}
