package handler

import (
	"fmt"
	"net/http"
)

func (h *AppHandler) signInURL() string {
	return fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", h.FirebaseAPIKey)
}

// refreshURL builds the Firebase REST refresh token URL using the injected API key.
func (h *AppHandler) refreshURL() string {
	return fmt.Sprintf("https://securetoken.googleapis.com/v1/token?key=%s", h.FirebaseAPIKey)
}

// signUpURL builds the Firebase REST sign-up URL using the injected API key.
func (h *AppHandler) signUpURL() string {
	return fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signUp?key=%s", h.FirebaseAPIKey)
}

// updateURL builds the Firebase REST account update URL using the injected API key.
func (h *AppHandler) updateURL() string {
	return fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:update?key=%s", h.FirebaseAPIKey)
}

// signInWithIdpURL builds the Firebase REST identity provider sign-in URL.
func (h *AppHandler) signInWithIdpURL() string {
	return fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithIdp?key=%s", h.FirebaseAPIKey)
}
func (h *AppHandler) GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	h.SocialLoginWithProvider(w, r, "google.com")
}

// FacebookLoginHandler xử lý đăng nhập qua Facebook.
func (h *AppHandler) FacebookLoginHandler(w http.ResponseWriter, r *http.Request) {
	h.SocialLoginWithProvider(w, r, "facebook.com")
}

// AppleLoginHandler xử lý đăng nhập qua Apple.
func (h *AppHandler) AppleLoginHandler(w http.ResponseWriter, r *http.Request) {
	h.SocialLoginWithProvider(w, r, "apple.com")
}