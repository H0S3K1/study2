package models

type Product struct {
	ID    string  `json:"id"`
	Brand string  `json:"brand"`
	CPU   string  `json:"cpu"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Type  string  `json:"type"`
}

type PingType struct {
	ID    string `json:"id" firestore:"-"`
	Type  string `json:"type" firestore:"type,omitempty"`
	Brand string `json:"brand" firestore:"brand,omitempty"`
	Name  string `json:"name" firestore:"name,omitempty"`
	CPU   string `json:"cpu" firestore:"cpu,omitempty"`
	//Images    []string `json:"images" firestore:"images,omitempty"`
	//Price     int      `json:"price" firestore:"price,omitempty"`
	//Storage   string   `json:"storage" firestore:"storage,omitempty"`
	//Thumbnail string   `json:"thumbnail" firestore:"thumbnail,omitempty"`
}

type Require struct {
	Field string
	Value string
}

type ProductFilter struct {
	Brands   []string `json:"brands"`
	CPUs     string   `json:"cpus"`
	MinPrice int      `json:"min_price" validate:"gte=0"`
	MaxPrice int      `json:"max_price" validate:"gte=0,gtefield=MinPrice"`
	Type     []string `json:"type"`
	// Thêm các trường khác nếu cần
}
