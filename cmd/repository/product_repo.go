package repository

import (
	"context"
	"study2/cmd/db"
	"study2/cmd/models"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// ProductRepository handles all Firestore + cache operations for products.
type ProductRepository struct {
	DB *firestore.Client
}

// FetchByIDs retrieves a batch of products from cache (if available) or Firestore.
// It returns the products and a list of IDs that were pulled fresh from Firestore.
func (r *ProductRepository) FetchByIDs(ctx context.Context, ids []string) ([]models.Product, error) {
	var products []models.Product
	var missingIDs []string

	for _, id := range ids {
		if p, found := db.MyCache.Get("prod:" + id); found {
			products = append(products, p.(models.Product))
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) > 0 {
		var docRefs []*firestore.DocumentRef
		for _, id := range missingIDs {
			docRefs = append(docRefs, r.DB.Collection("products").Doc(id))
		}

		docs, err := r.DB.GetAll(ctx, docRefs)
		if err != nil {
			return products, err
		}
		for _, doc := range docs {
			if !doc.Exists() {
				continue
			}
			var p models.Product
			if err := doc.DataTo(&p); err != nil {
				continue
			}
			p.ID = doc.Ref.ID
			products = append(products, p)
			db.MyCache.Set("prod:"+p.ID, p, 5*time.Hour)
		}
	}

	return products, nil
}

// FetchPage retrieves a paginated list of products ordered by price.
// It also collects unique brand names from the returned products.
func (r *ProductRepository) FetchPage(ctx context.Context, lastDocID string, limit int) (products []models.Product, brands []string, err error) {
	q := r.DB.Collection("products").OrderBy("price", firestore.Asc).Limit(limit)
	if lastDocID != "" {
		docSnap, _ := r.DB.Collection("products").Doc(lastDocID).Get(ctx)
		if docSnap != nil {
			q = q.StartAfter(docSnap)
		}
	}

	pIter := q.Documents(ctx)
	defer pIter.Stop()
	for {
		doc, docErr := pIter.Next()
		if docErr == iterator.Done {
			break
		}
		if docErr != nil {
			return nil, nil, docErr
		}

		var p models.Product
		if err := doc.DataTo(&p); err == nil {
			p.ID = doc.Ref.ID
			products = append(products, p)
		}
	}

	brandMap := make(map[string]bool)
	allDocsIter := r.DB.Collection("products").Select("brand").Documents(ctx)
	defer allDocsIter.Stop()

	for {
		doc, docErr := allDocsIter.Next()
		if docErr == iterator.Done {
			break
		}
		if docErr != nil {
			break
		}
		data := doc.Data()
		if bName, ok := data["brand"].(string); ok && bName != "" {
			if !brandMap[bName] {
				brandMap[bName] = true
				brands = append(brands, bName)
			}
		}
	}

	return products, brands, nil
}

// FetchByFilter retrieves products matching the given filter, with cursor pagination.
func (r *ProductRepository) FetchByFilter(ctx context.Context, filter models.ProductFilter, lastDocID string, limit int) ([]models.Product, error) {

	q := r.DB.Collection("products").Query

	// 1. Where filters first
	if len(filter.Brands) > 0 && filter.Brands[0] != "" {
		q = q.Where("brand", "in", filter.Brands)
	}
	if len(filter.Type) > 0 && filter.Type[0] != "" {
		q = q.Where("type", "in", filter.Type)
	}
	if filter.MinPrice > 0 {
		q = q.Where("price", ">=", filter.MinPrice)
	}
	if filter.MaxPrice > 0 {
		q = q.Where("price", "<=", filter.MaxPrice)
	}
	orderDir := firestore.Desc
	if filter.Sort == "asc" {
		orderDir = firestore.Asc
	}
	q = q.OrderBy("price", orderDir)
	// 3. Pagination and Limit last
	if lastDocID != "" {
		docSnap, err := r.DB.Collection("products").Doc(lastDocID).Get(ctx)
		if err == nil {
			q = q.StartAfter(docSnap)
		}
	}

	q = q.Limit(limit)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var products []models.Product
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var p models.Product
		if err := doc.DataTo(&p); err != nil {
			continue
		}
		p.ID = doc.Ref.ID
		products = append(products, p)
		db.MyCache.Set("prod:"+p.ID, p, 5*time.Hour)
	}
	return products, nil
}
