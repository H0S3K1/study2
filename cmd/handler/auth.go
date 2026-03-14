package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"study2/cmd/models"
	"study2/cmd/utils"
	"time"

	"cloud.google.com/go/firestore"
)

// signInURL builds the Firebase REST sign-in URL using the injected API key.
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
	var req models.RefreshRequest
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

	var req models.LoginRequest
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

	var loginResp models.LoginResponse
	if !callFirebaseREST(w, h.signInURL(), payload, &loginResp, "Sai email hoặc mật khẩu") {
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResp)
}

// RegisterHandler xử lý việc đăng ký user mới thông qua Firebase REST API
func (h *AppHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req models.RegisterRequest
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

	var regResp models.RegisterResponse
	if !callFirebaseREST(w, h.signUpURL(), payload, &regResp, "Đăng ký thất bại (có thể email đã được sử dụng)") {
		return
	}

	// Save user profile to Firestore using the UID as document ID
	_, err := h.DB.Collection("users").Doc(regResp.LocalID).Set(r.Context(), map[string]interface{}{
		"email":      regResp.Email,
		"created_at": time.Now(),
		"role":       "customer",
	})
	if err != nil {
		// Firebase Auth already created the account; log the DB error but don't block the response
		fmt.Printf("Lỗi tạo user profile trong Firestore: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(regResp)
}

// EditProfileHandler xử lý việc đổi mật khẩu hoặc email của user đang đăng nhập
func (h *AppHandler) EditProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	authHeader := r.Header.Get("Authorization")
	idToken := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		idToken = authHeader[7:]
	}

	var req models.EditProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Dữ liệu không hợp lệ", http.StatusBadRequest)
		return
	}

	if req.Email == "" && req.Password == "" {
		utils.SendJSONError(w, "Bạn chưa nhập Email hay Password mới nào để đổi", http.StatusBadRequest)
		return
	}

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

	var updateResp models.RegisterResponse
	if !callFirebaseREST(w, h.updateURL(), payload, &updateResp, "Thay đổi thất bại (token có thể đã hết hạn hoặc email đã được dùng)") {
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updateResp)
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

// UpdateProfileHandler cập nhật thông tin cá nhân của user trong Firestore
func (h *AppHandler) UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		utils.SendJSONError(w, "Không lấy được thông tin xác thực", http.StatusUnauthorized)
		return
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.SendJSONError(w, "Dữ liệu không hợp lệ", http.StatusBadRequest)
		return
	}

	docSnap, err := h.DB.Collection("users").Doc(userID).Get(ctx)
	if err != nil {
		utils.SendJSONError(w, "Không tìm thấy người dùng", http.StatusNotFound)
		return
	}

	currentData := docSnap.Data()
	updates := make(map[string]interface{})
	hasChanges := false

	compareAndAdd := func(field string, newValue interface{}) {
		switch v := newValue.(type) {
		case string:
			if v == "" {
				return
			}
		case int:
			if v == 0 {
				return
			}
		}
		oldValue, exists := currentData[field]
		if !exists || oldValue != newValue {
			hasChanges = true
			updates[field] = newValue
		}
	}

	compareAndAdd("name", req.Name)
	compareAndAdd("age", req.Age)
	compareAndAdd("address", req.Address)
	compareAndAdd("gender", req.Gender)
	compareAndAdd("phone_number", req.PhoneNumber)

	if !hasChanges {
		utils.SendJSONError(w, "The update is duplicated (Thông tin không thay đổi)", http.StatusConflict)
		return
	}

	_, err = h.DB.Collection("users").Doc(userID).Set(ctx, updates, firestore.MergeAll)
	if err != nil {
		utils.SendJSONError(w, "Lỗi cập nhật Firestore", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Cập nhật hồ sơ thành công",
	})
}

// LogoutHandler xử lý logout bằng cách yêu cầu client xóa token
func (h *AppHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	utils.SendJSONSuccess(w, map[string]string{
		"message": "Đăng xuất thành công. Client vui lòng xóa token hiện tại.",
	}, http.StatusOK)
}
