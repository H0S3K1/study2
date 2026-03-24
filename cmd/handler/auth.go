package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"study2/cmd/models"
	"study2/cmd/utils"
	"strings"
	"time"
)

// SocialLoginHandler xử lý đăng nhập qua Facebook, Google, Apple.
func (h *AppHandler) SocialLoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req models.SocialLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Vui lòng cung cấp ProviderID", http.StatusBadRequest)
		return
	}

	// Prepare payload for signInWithIdp
	var postBody string
	if req.IDToken != "" {
		postBody = fmt.Sprintf("id_token=%s&providerId=%s", req.IDToken, req.ProviderID)
	} else if req.AccessToken != "" {
		postBody = fmt.Sprintf("access_token=%s&providerId=%s", req.AccessToken, req.ProviderID)
	} else {
		utils.SendJSONError(w, "Vui lòng cung cấp idToken hoặc accessToken", http.StatusBadRequest)
		return
	}

	payload := map[string]interface{}{
		"postBody":            postBody,
		"requestUri":          "http://localhost", // Mandatory field for Firebase REST API
		"returnSecureToken":   true,
		"returnIdpCredential": true,
	}

	var loginResp models.AuthResponse
	if !callFirebaseREST(w, h.signInWithIdpURL(), payload, &loginResp, "Xác thực với "+req.ProviderID+" thất bại") {
		return
	}

	// Check if this is a new user, then we might want to save profile to Firestore
	// However, Firebase doesn't directly return if it's a new user here in simple response.
	// We can check if the user doc exists in Firestore.
	docRef := h.DB.Collection("users").Doc(loginResp.LocalID)
	docSnap, err := docRef.Get(r.Context())
	if err != nil {
		// If user doesn't exist, create profile
		_, err = docRef.Set(r.Context(), map[string]interface{}{
			"email":      loginResp.Email,
			"created_at": time.Now(),
			"role":       "customer",
		})
		if err != nil {
			fmt.Printf("Lỗi tạo user profile trong Firestore: %v\n", err)
		}
	} else {
		// Check if email or other info needs update if needed
		_ = docSnap
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResp)
}

// SocialLoginWithProvider là hàm generic hỗ trợ đăng nhập social.
func (h *AppHandler) SocialLoginWithProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	w.Header().Set("Content-Type", "application/json")

	var req models.SocialLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}
	req.ProviderID = providerID // Override providerID based on route

	loginResp, err := h.AuthRepo.SocialLogin(r.Context(), req)
	if err != nil {
		utils.SendJSONError(w, "Xác thực với "+providerID+" thất bại: "+err.Error(), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResp)
}

// callFirebaseREST is a helper for all Firebase Authentication REST API calls.
func callFirebaseREST(w http.ResponseWriter, url string, payload map[string]interface{}, successResp interface{}, errorMsg string) bool {
	payloadBytes, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		utils.SendJSONError(w, "Không kết nối được với Firebase Authentication", http.StatusInternalServerError)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.SendJSONError(w, errorMsg, http.StatusUnauthorized)
		return false
	}

	if err := json.NewDecoder(resp.Body).Decode(successResp); err != nil {
		utils.SendJSONError(w, "Lỗi đọc dữ liệu từ Firebase", http.StatusInternalServerError)
		return false
	}

	return true
}
func (h *AppHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req models.RefreshResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Vui lòng nhập Refresh Token hợp lệ", http.StatusBadRequest)
		return
	}
	payload := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": req.RefreshToken,
	}
	var refreshResp models.RefreshResponse
	if !callFirebaseREST(w, h.refreshURL(), payload, &refreshResp, "Refresh token không hợp lệ") {
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(refreshResp)
}

// LoginHandler xử lý việc đăng nhập của user thông qua Firebase REST API
func (h *AppHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Vui lòng nhập Email và Password hợp lệ", http.StatusBadRequest)
		return
	}

	payload := map[string]interface{}{
		"email":             req.Email,
		"password":          req.Password,
		"returnSecureToken": true,
	}

	var loginResp models.AuthResponse
	if !callFirebaseREST(w, h.signInURL(), payload, &loginResp, "Sai email hoặc mật khẩu") {
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResp)
}

// RegisterHandler xử lý việc đăng ký user mới thông qua Firebase REST API
func (h *AppHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Vui lòng cung cấp Email hợp lệ và mật khẩu ít nhất 6 ký tự", http.StatusBadRequest)
		return
	}

	payload := map[string]interface{}{
		"email":             req.Email,
		"password":          req.Password,
		"returnSecureToken": true,
	}

	var regResp models.AuthResponse
	if !callFirebaseREST(w, h.signUpURL(), payload, &regResp, "Đăng ký thất bại (có thể email đã được sử dụng)") {
		return
	}

	// Save user profile to Firestore using the UID as document ID
	regResp, err := h.AuthRepo.Create(r.Context(),regResp.LocalID, req.Email)
	if err != nil {
		// Firebase Auth already created the account; log the DB error but don't block the response
		fmt.Printf("Lỗi tạo user profile trong Firestore: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(regResp)
}



// EditProfileHandler xử lý việc đổi mật khẩu hoặc email của user đang đăng nhập
func (h *AppHandler) UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := r.Context().Value("user_id").(string)

	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi JSON", http.StatusBadRequest)
		return
	}

	// 1. Update Firebase Auth (Email/Password) if provided
	if req.Email != "" || req.Password != "" {
		authHeader := r.Header.Get("Authorization")
		idToken := strings.TrimPrefix(authHeader, "Bearer ")

		payload := map[string]interface{}{
			"idToken":           idToken,
			"returnSecureToken": true,
		}
		if req.Email != "" {
			payload["email"] = req.Email
		}
		if req.Password != "" {
			payload["password"] = req.Password
		}

		var authResp models.AuthResponse
		if !callFirebaseREST(w, h.updateURL(), payload, &authResp, "Cập nhật Firebase Auth thất bại") {
			return
		}
	}

	// 2. Update Firestore Profile (Name, Age, etc.)
	var reqs models.ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		utils.SendJSONError(w, "Lỗi JSON", http.StatusBadRequest)
		return
	}
	_,err := h.AuthRepo.Update(r.Context(),reqs , userID)
	if err != nil {
		utils.SendJSONError(w, "Cập nhật Firestore thất bại", http.StatusInternalServerError)
		return
	}

	utils.SendJSONSuccess(w, map[string]string{"message": "Cập nhật thành công"}, http.StatusOK)
}

// GetProfileHandler lấy thông tin profile của user từ Firestore
func (h *AppHandler) GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.SendJSONError(w, "Không lấy được thông tin xác thực của user", http.StatusUnauthorized)
		return
	}

	docSnap, err := h.DB.Collection("users").Doc(userID).Get(r.Context())
	if err != nil {
		utils.SendJSONError(w, "Không tìm thấy user profile trong hệ thống", http.StatusNotFound)
		return
	}

	var user models.User
	if err := docSnap.DataTo(&user); err != nil {
		utils.SendJSONError(w, "Lỗi khi đọc dữ liệu profile", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// LogoutHandler xử lý logout bằng cách yêu cầu client xóa token
func (h *AppHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	utils.SendJSONSuccess(w, map[string]string{
		"message": "Đăng xuất thành công. Client vui lòng xóa token hiện tại.",
	}, http.StatusOK)
}
