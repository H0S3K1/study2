package handler

import (
	"log"
	"net/http"
	"study2/internal/models"
	"study2/internal/utils"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// 1. Tạo Struct chứa Client (Đây là cốt lõi của Dependency Injection)

// 2. Biến hàm thành Method của AppHandler (Có cái (h *AppHandler) ở đằng trước)
// Chú ý: Đã bỏ cái http.HandleFunc lồng bên trong đi
func (h *AppHandler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var products []models.Product

	limit := 20
	lastDocID := r.URL.Query().Get("lastDocID")

	q := h.DB.Collection("products").OrderBy("price", firestore.Asc).Limit(limit)

	if lastDocID != "" {
		docSnap, err := h.DB.Collection("products").Doc(lastDocID).Get(r.Context())
		if err == nil {
			q = q.StartAfter(docSnap)
		}
	}

	// 3. Sử dụng h.DB thay vì biến client vô danh
	iter := q.Documents(r.Context())

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
		products = append(products, p)
	}

	utils.SendJSONSuccess(w, products, http.StatusOK)
}
