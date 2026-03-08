package models

type CartItem struct {
	ProductID string  `json:"product_id" firestore:"product_id" validate:"required"`
	Quantity  int     `json:"quantity" firestore:"quantity" validate:"required"`
	Price     float64 `json:"price" firestore:"price" validate:"required"`
	Name      string  `json:"name" firestore:"name" validate:"required"`
	Thumbnail string  `json:"thumbnail" firestore:"thumbnail" validate:"required"`
}

type AddToCartRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}
