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
	var req models.CartItem
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

	// 6. Thêm vào sub-collection "cart" (tự động tạo mới nếu chưa có)
	cartRef := h.DB.Collection("users").Doc(uid).Collection("cart").Doc(req.ProductID)
	_, err = cartRef.Set(ctx, cartItem)
	if err != nil {
		utils.SendJSONError(w, "Lỗi cập nhật giỏ hàng", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Đã thêm sản phẩm vào giỏ hàng"})
}

func (h *AppHandler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	uid := ctx.Value("user_id").(string)

	var req models.RemoveFromCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Dữ liệu không hợp lệ", http.StatusBadRequest)
		return
	}

	cartRef := h.DB.Collection("users").Doc(uid).Collection("cart").Doc(req.ProductID)
	_, err := cartRef.Delete(ctx)

	if err != nil {
		utils.SendJSONError(w, "Lỗi xóa sản phẩm khỏi giỏ hàng", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Đã xóa sản phẩm khỏi giỏ hàng"})
}

func (h *AppHandler) UpdateCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	uid := ctx.Value("user_id").(string)

	var req models.AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Dữ liệu không hợp lệ", http.StatusBadRequest)
		return
	}

	cartRef := h.DB.Collection("users").Doc(uid).Collection("cart").Doc(req.ProductID)
	_, err := cartRef.Update(ctx, []firestore.Update{
		{
			Path:  "quantity",
			Value: req.Quantity,
		},
	})

	if err != nil {
		utils.SendJSONError(w, "Sản phẩm không có trong giỏ hàng hoặc lỗi cập nhật", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Đã cập nhật số lượng sản phẩm"})
}

func (h *AppHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	uid := ctx.Value("user_id").(string)

	cartRef := h.DB.Collection("users").Doc(uid).Collection("cart")
	docSnaps, err := cartRef.Documents(ctx).GetAll()
	if err != nil {
		utils.SendJSONError(w, "Lỗi khi lấy giỏ hàng", http.StatusInternalServerError)
		return
	}

	var cart []models.CartItem
	for _, snap := range docSnaps {
		var item models.CartItem
		if err := snap.DataTo(&item); err != nil {
			continue
		}
		cart = append(cart, item)
	}

	if cart == nil {
		cart = []models.CartItem{}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cart)
}
