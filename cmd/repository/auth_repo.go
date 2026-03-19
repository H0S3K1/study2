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

func (r *AuthRepository) Login(ctx context.Context, req models.LoginRequest) (models.LoginResponse, error) {
	var resp models.LoginResponse
	return resp, nil
}

func (r *AuthRepository) SocialLogin(ctx context.Context, req models.SocialLoginRequest) (models.LoginResponse, error) {
	var postBody string
	if req.IDToken != "" {
		postBody = fmt.Sprintf("id_token=%s&providerId=%s", req.IDToken, req.ProviderID)
	} else if req.AccessToken != "" {
		postBody = fmt.Sprintf("access_token=%s&providerId=%s", req.AccessToken, req.ProviderID)
	} else {
		return models.LoginResponse{}, fmt.Errorf("idToken or accessToken required")
	}

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithIdp?key=%s", r.FirebaseAPIKey)
	payload := map[string]interface{}{
		"postBody":           postBody,
		"requestUri":          "http://localhost",
		"returnSecureToken":   true,
		"returnIdpCredential": true,
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return models.LoginResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return models.LoginResponse{}, fmt.Errorf("firebase auth error: %v", errResp)
	}

	var loginResp models.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return models.LoginResponse{}, err
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
