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
	brandMap := make(map[string]bool)

	q := r.DB.Collection("products").OrderBy("price", firestore.Asc).Limit(limit)
	if lastDocID != "" {
		docSnap, snapErr := r.DB.Collection("products").Doc(lastDocID).Get(ctx)
		if snapErr == nil {
			q = q.StartAfter(docSnap)
		}
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	for {
		doc, docErr := iter.Next()
		if docErr == iterator.Done {
			break
		}
		if docErr != nil {
			err = docErr
			return
		}
		var p models.Product
		if mapErr := doc.DataTo(&p); mapErr != nil {
			continue
		}
		p.ID = doc.Ref.ID
		products = append(products, p)
		db.MyCache.Set("prod:"+p.ID, p, 5*time.Hour)

		if p.Brand != "" && !brandMap[p.Brand] {
			brandMap[p.Brand] = true
			brands = append(brands, p.Brand)
		}
	}
	return
}

// FetchByFilter retrieves products matching the given filter, with cursor pagination.
func (r *ProductRepository) FetchByFilter(ctx context.Context, filter models.ProductFilter, lastDocID string, limit int) ([]models.Product, error) {
	q := r.DB.Collection("products").OrderBy("price", firestore.Asc).Limit(limit)

	if lastDocID != "" {
		docSnap, err := r.DB.Collection("products").Doc(lastDocID).Get(ctx)
		if err == nil {
			q = q.StartAfter(docSnap)
		}
	}

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
