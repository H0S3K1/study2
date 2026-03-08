package handler

import (
	"encoding/json"
	"net/http"
	"study2/internal/models" // Nhớ check lại đường dẫn import của ông
	"study2/internal/utils"

	"cloud.google.com/go/firestore"
)

func (h *AppHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Lấy thông tin user đang đăng nhập
	ctx := r.Context()
	uid := ctx.Value("user_id").(string)

	// 2. Đọc request body
	var req models.AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	// 3. Validate request
	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Dữ liệu không hợp lệ", http.StatusBadRequest)
		return
	}

	// 4. Lấy thông tin sản phẩm từ Firestore để kiểm tra và lấy giá/ảnh
	productRef := h.DB.Collection("products").Doc(req.ProductID)
	productSnap, err := productRef.Get(r.Context())
	if err != nil {
		utils.SendJSONError(w, "Không tìm thấy sản phẩm", http.StatusNotFound)
		return
	}

	var product models.Product
	if err := productSnap.DataTo(&product); err != nil {
		utils.SendJSONError(w, "Lỗi đọc dữ liệu sản phẩm", http.StatusInternalServerError)
		return
	}

	cartItem := models.CartItem{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Price:     product.Price,
		Name:      product.Name,
	}
	if len(product.Images) > 0 {
		cartItem.Thumbnail = product.Images[0]
	}

	// 6. Cập nhật giỏ hàng của user
	userRef := h.DB.Collection("users").Doc(uid)
	_, err = userRef.Update(r.Context(), []firestore.Update{
		{
			Path:  "cart",
			Value: firestore.ArrayUnion(cartItem),
		},
	})
	if err != nil {
		utils.SendJSONError(w, "Lỗi cập nhật giỏ hàng", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Đã thêm sản phẩm vào giỏ hàng"})
}
