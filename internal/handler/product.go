package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"study2/internal/db"
	"study2/internal/models" // Nhớ check lại đường dẫn import của ông
	"study2/internal/utils"
	"time"

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

	// Create a unique cache key for this page
	cacheKey := "products_page_"
	if lastDocID != "" {
		cacheKey += lastDocID
	} else {
		cacheKey += "first"
	}

	// Check the cache
	if cachedData, found := db.MyCache.Get(cacheKey); found {
		utils.SendJSONSuccess(w, cachedData, http.StatusOK)
		return
	}

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

	// Save to cache before returning
	db.MyCache.Set(cacheKey, products, 5*time.Minute)
	db.AddSuggestion(cacheKey)
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

	// Check cache
	cacheKey := "product_id_" + id
	if cachedData, found := db.MyCache.Get(cacheKey); found {
		utils.SendJSONSuccess(w, cachedData, http.StatusOK)
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

	// Save to cache
	db.MyCache.Set(cacheKey, product, 5*time.Minute)

	utils.SendJSONSuccess(w, product, http.StatusOK)
}
func (h *AppHandler) SeachingProd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	queryLower := r.URL.Query().Get("query")
	if queryLower == "" {
		utils.SendJSONError(w, "Thiếu query", http.StatusBadRequest)
		return
	}
	query := strings.ToLower(queryLower)
	cacheKey := "search_" + query

	// Check cache
	if cachedData, found := db.MyCache.Get(cacheKey); found {
		utils.SendJSONSuccess(w, cachedData, http.StatusOK)
		return
	}

	iter := h.DB.Collection("products").
		//Where("name_lowercase", ">=", query).
		//Where("name_lowercase", "<", query+"\uf8ff").
		Limit(20). // Chỉ lấy 20 thằng thôi, không ai xem hết 1000 món một lúc
		Documents(r.Context())
	defer iter.Stop()
	var prods []models.Product
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
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Brand), query) ||
			strings.Contains(strings.ToLower(p.Type), query) {
			prods = append(prods, p)
		}
	}
	db.AddSuggestion(query)
	// Save to cache
	db.MyCache.Set(cacheKey, prods, 10*time.Minute)
	utils.SendJSONSuccess(w, prods, http.StatusOK)
}
