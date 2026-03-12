package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"study2/internal/models" // Nhớ check lại đường dẫn import của ông
	"study2/internal/utils"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

func (h *AppHandler) GetProductsByFilter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var payload models.ProductFilter
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}
	if err := utils.Validate.Struct(payload); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
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

func (h *AppHandler) GetProductByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var product models.Product

	// Lấy id từ path parameter (Go 1.22+)
	id := r.PathValue("id")
	if id == "" {
		utils.SendJSONError(w, "Thiếu ID sản phẩm", http.StatusBadRequest)
		return
	}

	docSnap, err := h.DB.Collection("products").Doc(id).Get(r.Context())
	if err != nil {
		utils.SendJSONError(w, "Không tìm thấy sản phẩm", http.StatusNotFound)
		return
	}

	if err := docSnap.DataTo(&product); err != nil {
		log.Printf("Lỗi map dữ liệu: %v", err)
		utils.SendJSONError(w, "Lỗi xử lý dữ liệu sản phẩm", http.StatusInternalServerError)
		return
	}

	product.ID = docSnap.Ref.ID
	utils.SendJSONSuccess(w, product, http.StatusOK)
}
func (h *AppHandler) SeachingProd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	query := r.URL.Query().Get("query")
	if query == "" {
		utils.SendJSONError(w, "Thiếu query", http.StatusBadRequest)
		return
	}

	queryLower := strings.ToLower(query)
	var products []models.Product

	iter := h.DB.Collection("products").Documents(r.Context())
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			utils.SendJSONError(w, "Lỗi truy vấn dữ liệu", http.StatusInternalServerError)
			return
		}

		var p models.Product
		if err := doc.DataTo(&p); err != nil {
			log.Printf("Lỗi map dữ liệu: %v", err)
			continue
		}
		p.ID = doc.Ref.ID

		// Find products where name or brand contains the query constraint
		if strings.Contains(strings.ToLower(p.Name), queryLower) || strings.Contains(strings.ToLower(p.Brand), queryLower) {
			products = append(products, p)
		}
	}

	utils.SendJSONSuccess(w, products, http.StatusOK)
}
