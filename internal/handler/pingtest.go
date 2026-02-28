package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"study2/internal/models"
	"study2/internal/utils"

	"google.golang.org/api/iterator"
)

// 1. Tạo Struct chứa Client (Đây là cốt lõi của Dependency Injection)

// 2. Biến hàm thành Method của AppHandler (Có cái (h *AppHandler) ở đằng trước)
// Chú ý: Đã bỏ cái http.HandleFunc lồng bên trong đi
func (h *AppHandler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var products []models.Product

	// 3. Sử dụng h.DB thay vì biến client vô danh
	iter := h.DB.Collection("products").Documents(r.Context())

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			utils.SendJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var p models.Product
		if err := doc.DataTo(&p); err != nil {
			log.Printf("Lỗi map dữ liệu: %v", err)
			continue
		}
		p.ID = doc.Ref.ID
		data, _ := json.MarshalIndent(p, "", " ")
		log.Printf("Lấy được product: %s", string(data))
		products = append(products, p)
	}

	utils.SendJSONSuccess(w, products, http.StatusOK)
}
