package repository

import (
	"bytes"
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"study2/cmd/models"
	"time"
)

type AuthRepository struct {
	DB             *firestore.Client
	FirebaseAPIKey string
}


func (r *AuthRepository) Create(ctx context.Context,userID, email string) (models.AuthResponse, error) {
	docRef := r.DB.Collection("users").Doc(userID)
	_, err := docRef.Get(ctx)
	if err != nil {
		_, err = docRef.Set(ctx, map[string]interface{}{
			"email":      email,
			"created_at": time.Now(),
			"role":       "customer",
		})
		if err != nil {
			return models.AuthResponse{}, err
		}
	}
	docRef.Set(ctx, map[string]interface{}{"email": email, "updated_at": time.Now(), "role": "customer"})
	return models.AuthResponse{} , nil
}

func (r *AuthRepository) Login(ctx context.Context, req models.AuthRequest) (models.AuthResponse, error) {
	var resp models.AuthResponse
	return resp, nil
}
func (r *AuthRepository) Update(ctx context.Context, req models.ProfileRequest , uid string) (models.AuthResponse, error) {
	update := []firestore.Update{}
	if req.Name != "" {
		update = append(update, firestore.Update{Path: "name", Value: req.Name})
	}
	if req.Age != 0 {
		update = append(update, firestore.Update{Path: "age", Value: req.Age})
	}
	if req.Address != "" {
		update = append(update, firestore.Update{Path: "address", Value: req.Address})
	}
	if req.Gender != "" {
		update = append(update, firestore.Update{Path: "gender", Value: req.Gender})
	}
	if req.PhoneNumber != "" {
		update = append(update, firestore.Update{Path: "phone_number", Value: req.PhoneNumber})
	}
	if len(update) == 0 {
		return models.AuthResponse{}, nil
	}
	update = append(update, firestore.Update{Path: "updated_at", Value: time.Now()})
	_,err := r.DB.Collection("users").Doc(uid).Update(ctx, update)
	return models.AuthResponse{}, err
}

func (r *AuthRepository) SocialLogin(ctx context.Context, req models.SocialLoginRequest) (models.AuthResponse, error) {
	var postBody string
	if req.IDToken != "" {
		postBody = fmt.Sprintf("id_token=%s&providerId=%s", req.IDToken, req.ProviderID)
	} else if req.AccessToken != "" {
		postBody = fmt.Sprintf("access_token=%s&providerId=%s", req.AccessToken, req.ProviderID)
	} else {
		return models.AuthResponse{}, fmt.Errorf("idToken or accessToken required")
	}

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithIdp?key=%s", r.FirebaseAPIKey)
	payload := map[string]interface{}{
		"postBody":            postBody,
		"requestUri":          "http://localhost",
		"returnSecureToken":   true,
		"returnIdpCredential": true,
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return models.AuthResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return models.AuthResponse{}, fmt.Errorf("firebase auth error: %v", errResp)
	}

	var loginResp models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return models.AuthResponse{}, err
	}

	// Create profile if not exist
	docRef := r.DB.Collection("users").Doc(loginResp.LocalID)
	_, err = docRef.Get(ctx)
	if err != nil {
		_, _ = docRef.Set(ctx, map[string]interface{}{
			"email":      loginResp.Email,
			"created_at": time.Now(),
			"role":       "customer",
		})
	}

	return loginResp, nil
}
