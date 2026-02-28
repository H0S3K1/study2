package handler

import (
	"cloud.google.com/go/firestore"
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"google.golang.org/api/iterator"
	"log"
	"net/http"
	"study2/internal/models" // Nhớ check lại đường dẫn import của ông
)

var validate = validator.New()

func (h *AppHandler) GetProductByFillter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var payload models.ProductFilter
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}
	if err := validate.Struct(payload); err != nil {
		http.Error(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}
	// Pagination
	limit := 5
	lastDocID := r.URL.Query().Get("lastDocID") // Client truyền lên ID của item cuối trang trước

	q := h.DB.Collection("products").OrderBy("price", firestore.Asc).Limit(limit)

	// Nếu có lastDocID, mình phải query lấy cái snapshot của nó ra trước để làm mốc
	if lastDocID != "" {
		docSnap, err := h.DB.Collection("products").Doc(lastDocID).Get(r.Context())
		if err == nil {
			q = q.StartAfter(docSnap) // Bắt đầu từ sau doc này
		}
	}

	// 4. Bắt đầu nhồi điều kiện (Multi Filter)
	if len(payload.Brands) > 0 && payload.Brands[0] != "" {
		q = q.Where("brand", "in", payload.Brands)
	}
	if payload.CPUs != "" {
		// Chỉ được dùng 1 toán tử 'IN' trong 1 query.
		// Nếu đã dùng IN cho brand, thì cpu không được dùng IN nữa. (Phải xử lý bằng code Go như đã bàn)
		q = q.Where("cpu", "in", payload.CPUs)
	}
	if len(payload.Type) > 0 && payload.Type[0] != "" {
		q = q.Where("type", "in", payload.Type)
	}
	if payload.MinPrice > 0 {
		q = q.Where("price", ">=", payload.MinPrice)
	}
	if payload.MaxPrice > 0 {
		q = q.Where("price", "<=", payload.MaxPrice)
	}

	iter := q.Documents(r.Context())
	var products []models.Product
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var p models.Product
		if err := doc.DataTo(&p); err != nil {
			log.Printf("Lỗi map dữ liệu: %v", err)
			continue
		}
		p.ID = doc.Ref.ID
		log.Printf("Lấy được product: %s, ID: %s", p.Name, p.ID)
		products = append(products, p)
	}
	json.NewEncoder(w).Encode(products)
}
