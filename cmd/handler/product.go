package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"study2/cmd/db"
	"study2/cmd/models"
	"study2/cmd/utils"
	"time"

	"google.golang.org/api/iterator"
)

func (h *AppHandler) GetProductsByFilter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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

	// 2. Build cache key
	lastDocID := r.URL.Query().Get("lastDocID")
	cacheKey := fmt.Sprintf("filter:last:%s:brands:%v:types:%v:min:%d:max:%d:sort:%s",
		lastDocID, payload.Brands, payload.Type, payload.MinPrice, payload.MaxPrice, payload.Sort)

	// 3. Cache Hit — reconstruct product list from stored IDs
	if cachedData, found := db.MyCache.Get(cacheKey); found {
		prodIDs := cachedData.([]string)
		products, err := h.Repo.FetchByIDs(r.Context(), prodIDs)
		if err != nil {
			utils.SendJSONError(w, "Lỗi lấy dữ liệu sản phẩm", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(products)
		return
	}

	// 4. Cache Miss — query Firestore via repository
	products, err := h.Repo.FetchByFilter(r.Context(), payload, lastDocID, 20)
	if err != nil {
		utils.SendJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Cache result IDs and return
	var ids []string
	for _, p := range products {
		ids = append(ids, p.ID)
	}
	db.MyCache.Set(cacheKey, ids, 1*time.Hour)
	json.NewEncoder(w).Encode(products)
}

func (h *AppHandler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	lastDocID := r.URL.Query().Get("lastDocID")
	cacheKey := "products_page_"
	if lastDocID != "" {
		cacheKey += lastDocID
	} else {
		cacheKey += "first"
	}
	
	// Cache Hit
	if cachedData, found := db.MyCache.Get(cacheKey); found {
		cacheMap := cachedData.(map[string]interface{})
		prodIDs := cacheMap["ids"].([]string)
		uniqueBrands := cacheMap["brands"].([]string)

		products, err := h.Repo.FetchByIDs(r.Context(), prodIDs)
		if err != nil {
			utils.SendJSONError(w, "Lỗi lấy dữ liệu sản phẩm", http.StatusInternalServerError)
			return
		}

		utils.SendJSONSuccess(w, map[string]interface{}{
			"products": products,
			"brands":   uniqueBrands,
		}, http.StatusOK)
		return
	}

	// Cache Miss — fetch from Firestore via repository
	products, brands, err := h.Repo.FetchPage(r.Context(), lastDocID, 20)
	if err != nil {
		utils.SendJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if brands == nil {
		brands = []string{}
	}
	if products == nil {
		products = []models.Product{}
	}

	var storedIDs []string
	for _, p := range products {
		storedIDs = append(storedIDs, p.ID)
	}
	db.MyCache.Set(cacheKey, map[string]interface{}{
		"ids":    storedIDs,
		"brands": brands,
	}, 5*time.Hour)

	utils.SendJSONSuccess(w, map[string]interface{}{
		"products": products,
		"brands":   brands,
	}, http.StatusOK)
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

	iter := h.DB.Collection("products").Documents(r.Context())
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

	// Only track the clean human-readable query as a suggestion
	db.AddSuggestion(query)
	db.MyCache.Set(cacheKey, prods, 1*time.Hour)
	utils.SendJSONSuccess(w, prods, http.StatusOK)
}
