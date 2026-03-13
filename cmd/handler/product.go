package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"study2/cmd/db"
	"study2/cmd/models" // Nhớ check lại đường dẫn import của ông
	"study2/cmd/utils"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

func (h *AppHandler) GetProductsByFilter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var prodID []string
	var payload models.ProductFilter

	// 1. Decode and Validate Input
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}
	if err := utils.Validate.Struct(payload); err != nil {
		utils.SendJSONError(w, "Dữ liệu không hợp lệ", http.StatusBadRequest)
		return
	}

	// 2. Build the COMPLETE Cache Key first
	lastDocID := r.URL.Query().Get("lastDocID")
	cacheKey := fmt.Sprintf("filter:last:%s:brands:%v:types:%v:min:%d:max:%d",
		lastDocID, payload.Brands, payload.Type, payload.MinPrice, payload.MaxPrice)

	// 3. Check Cache (Hit)
	var total []models.Product
	if cachedData, found := db.MyCache.Get(cacheKey); found {
		prodID = cachedData.([]string)
		
		var missingIDs []string
		for _, id := range prodID {
			if p, found := db.MyCache.Get("prod:" + id); found {
				total = append(total, p.(models.Product))
			} else {
				missingIDs = append(missingIDs, id)
			}
		}

		if len(missingIDs) > 0 {
			// Fetch only the missing documents
			var docRefs []*firestore.DocumentRef
			for _, id := range missingIDs {
				docRefs = append(docRefs, h.DB.Collection("products").Doc(id))
			}
			
			docs, err := h.DB.GetAll(r.Context(), docRefs)
			if err == nil {
				for _, doc := range docs {
					if doc.Exists() {
						var p models.Product
						doc.DataTo(&p)
						p.ID = doc.Ref.ID
						total = append(total, p)
						db.MyCache.Set("prod:"+p.ID, p, 5*time.Hour)
					}
				}
			}
		}
		
		json.NewEncoder(w).Encode(total)
		return
	}

	// 4. Cache Miss - Build Firestore Query
	limit := 100
	q := h.DB.Collection("products").OrderBy("price", firestore.Asc).Limit(limit)

	// Handle Pagination Marker
	if lastDocID != "" {
		docSnap, err := h.DB.Collection("products").Doc(lastDocID).Get(r.Context())
		if err == nil {
			q = q.StartAfter(docSnap)
		}
	}

	// Add Filters
	if len(payload.Brands) > 0 && payload.Brands[0] != "" {
		q = q.Where("brand", "in", payload.Brands)
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

	// 5. Execute Query
	iter := q.Documents(r.Context())
	defer iter.Stop()

	var newProdIDs []string
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
			continue
		}
		p.ID = doc.Ref.ID
		total = append(total, p)
		newProdIDs = append(newProdIDs, p.ID)
		
		// Fill the individual product cache
		db.MyCache.Set("prod:"+p.ID, p, 5*time.Hour)
	}

	// 6. Save exactly to Cache and Return
	db.MyCache.Set(cacheKey, newProdIDs, 5*time.Hour)
	db.AddSuggestion(cacheKey)
	json.NewEncoder(w).Encode(total)
}

func (h *AppHandler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var products []models.Product
	var uniqueBrands []string
	brandMap := make(map[string]bool)

	limit := 20
	lastDocID := r.URL.Query().Get("lastDocID")
	// brands := r.URL.Query().Get("brands")

	// Create a unique cache key for this page
	cacheKey := "products_page_"
	if lastDocID != "" {
		cacheKey += lastDocID
	} else {
		cacheKey += "first"
	}

	// Check the cache
	if cachedData, found := db.MyCache.Get(cacheKey); found {
		cacheMap := cachedData.(map[string]interface{})
		prodIDs := cacheMap["ids"].([]string)
		uniqueBrands = cacheMap["brands"].([]string)

		var missingIDs []string
		for _, id := range prodIDs {
			if p, found := db.MyCache.Get("prod:" + id); found {
				products = append(products, p.(models.Product))
			} else {
				missingIDs = append(missingIDs, id)
			}
		}

		if len(missingIDs) > 0 {
			// Fetch only the missing documents
			var docRefs []*firestore.DocumentRef
			for _, id := range missingIDs {
				docRefs = append(docRefs, h.DB.Collection("products").Doc(id))
			}
			
			docs, err := h.DB.GetAll(r.Context(), docRefs)
			if err == nil {
				for _, doc := range docs {
					if doc.Exists() {
						var p models.Product
						doc.DataTo(&p)
						p.ID = doc.Ref.ID
						products = append(products, p)
						db.MyCache.Set("prod:"+p.ID, p, 5*time.Hour)
					}
				}
			}
		}

		response := map[string]interface{}{
			"products": products,
			"brands":   uniqueBrands,
		}
		utils.SendJSONSuccess(w, response, http.StatusOK)
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

		if p.Brand != "" && !brandMap[p.Brand] {
			brandMap[p.Brand] = true
			uniqueBrands = append(uniqueBrands, p.Brand)
		}
	}

	// Save products to individual cache and build IDs array
	var storedIDs []string
	for _, p := range products {
		storedIDs = append(storedIDs, p.ID)
		db.MyCache.Set("prod:"+p.ID, p, 5*time.Hour)
	}

	// Save to cache before returning
	if uniqueBrands == nil {
		uniqueBrands = []string{}
	}
	if products == nil {
		products = []models.Product{}
	}

	response := map[string]interface{}{
		"products": products,
		"brands":   uniqueBrands,
	}

	cacheStorage := map[string]interface{}{
		"ids":    storedIDs,
		"brands": uniqueBrands,
	}

	db.MyCache.Set(cacheKey, cacheStorage, 5*time.Hour)
	db.AddSuggestion(cacheKey)
	utils.SendJSONSuccess(w, response, http.StatusOK)
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
