package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"study2/internal/models"
	"study2/internal/utils"
	"time"
)

// helper function gửi json lỗi
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Helper để gọi Firebase Authentication REST API
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

// LoginHandler xử lý việc đăng nhập của user thông qua Firebase REST API
func (h *AppHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Đọc request body
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	// 2. Validate request (kiểm tra định dạng email, độ dài password)
	if err := validate.Struct(req); err != nil {
		sendJSONError(w, "Vui lòng nhập Email và Password hợp lệ", http.StatusBadRequest)
		return
	}

	// 3. Chuẩn bị payload để gọi Firebase Authentication REST API

	payload := map[string]interface{}{
		"email":             req.Email,
		"password":          req.Password,
		"returnSecureToken": true, // Quan trọng: Yêu cầu Firebase trả về JWT Token thật
	}

	// 4. Gửi HTTP POST Request sang Google và xử lý kết quả
	var loginResp models.LoginResponse
	if !callFirebaseREST(w, signInURL, payload, &loginResp, "Sai email hoặc mật khẩu") {
		return // Lỗi đã được xử lý trong helper
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResp)
}

// RegisterHandler xử lý việc đăng ký user mới thông qua Firebase REST API
func (h *AppHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Đọc request body
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	// 2. Validate request (kiểm tra định dạng email, độ dài password >= 6)
	if err := validate.Struct(req); err != nil {
		sendJSONError(w, "Vui lòng cung cấp Email hợp lệ và mật khẩu ít nhất 6 ký tự", http.StatusBadRequest)
		return
	}

	// 3. Chuẩn bị payload để gọi Firebase Authentication REST API (signUp)

	payload := map[string]interface{}{
		"email":             req.Email,
		"password":          req.Password,
		"returnSecureToken": true, // Yêu cầu trả về token luôn sau khi đăng ký thành công
	}

	// 4. Gửi HTTP POST Request sang Google và xử lý phản hồi
	var regResp models.RegisterResponse
	if !callFirebaseREST(w, signUpURL, payload, &regResp, "Đăng ký thất bại (có thể email đã được sử dụng)") {
		return
	}

	// 5. Lưu user profile (Filing Cabinet) vào Firestore!
	// Dùng LocalID (chính là UID của Firebase Auth) làm Document ID
	_, err := h.DB.Collection("users").Doc(regResp.LocalID).Set(r.Context(), map[string]interface{}{
		"email":      regResp.Email,
		"created_at": time.Now(),
		"role":       "customer", // Mặc định ai đăng ký cũng là customer
	})
	if err != nil {
		fmt.Printf("Lỗi tạo user profile trong Firestore: %v\n", err)
		// Dù lỗi ghi DB thì vẫn tiếp tục vì Firebase Auth đã tạo tài khoản thành công rồi
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(regResp)
}

// EditProfileHandler xử lý việc đổi mật khẩu hoặc email của user đang đăng nhập
func (h *AppHandler) EditProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Phải lấy lại idToken từ Header vì API thay đổi profile yêu cầu truyền idToken
	authHeader := r.Header.Get("Authorization")
	idToken := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		idToken = authHeader[7:]
	}

	// 2. Đọc request body
	var req models.EditProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}

	// 3. Validate request
	if err := validate.Struct(req); err != nil {
		sendJSONError(w, "Dữ liệu không hợp lệ", http.StatusBadRequest)
		return
	}

	// Nếu user không gửi email hay password gì để đổi thì báo lỗi
	if req.Email == "" && req.Password == "" {
		sendJSONError(w, "Bạn chưa nhập Email hay Password mới nào để đổi", http.StatusBadRequest)
		return
	}

	// 4. Chuẩn bị payload để gọi Firebase Authentication REST API (update)

	payload := map[string]interface{}{
		"idToken":           idToken, // Truyền token của user vào để chứng minh là chính chủ
		"returnSecureToken": true,
	}

	// Cập nhật những gì user yêu cầu
	if req.Email != "" {
		payload["email"] = req.Email
	}
	if req.Password != "" {
		payload["password"] = req.Password
	}

	// 5. Gọi HTTP POST Request sang Google
	var updateResp models.RegisterResponse // Dùng chung struct với register cho kết quả update
	if !callFirebaseREST(w, updateURL, payload, &updateResp, "Thay đổi thất bại (token có thể đã hết hạn hoặc email đã được dùng)") {
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updateResp)
}

// GetProfileHandler lấy thông tin profile của user từ Firestore
func (h *AppHandler) GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Lấy UID của user từ Context (do AuthMiddleware truyền vào)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		sendJSONError(w, "Không lấy được thông tin xác thực của user", http.StatusUnauthorized)
		return
	}

	// 2. Query trực tiếp vào Firestore dựa trên UID
	docSnap, err := h.DB.Collection("users").Doc(userID).Get(r.Context())
	if err != nil {
		sendJSONError(w, "Không tìm thấy user profile trong hệ thống", http.StatusNotFound)
		return
	}

	// 3. Map dữ liệu trả về từ Firestore vào struct models.User
	var user models.User
	if err := docSnap.DataTo(&user); err != nil {
		sendJSONError(w, "Lỗi khi đọc dữ liệu profile", http.StatusInternalServerError)
		return
	}

	// 4. Bắn về cho Client
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// UpdateProfileHandler cập nhật thông tin cá nhân của user trong Firestore
func (h *AppHandler) UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	// 1. Authenticate User
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		sendJSONError(w, "Không lấy được thông tin xác thực", http.StatusUnauthorized)
		return
	}

	// 2. Decode & Validate Request
	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Lỗi cú pháp JSON", http.StatusBadRequest)
		return
	}
	if err := validate.Struct(req); err != nil {
		sendJSONError(w, "Dữ liệu không hợp lệ", http.StatusBadRequest)
		return
	}

	// 3. Fetch Existing Data to Compare
	docSnap, err := h.DB.Collection("users").Doc(userID).Get(ctx)
	if err != nil {
		sendJSONError(w, "Không tìm thấy người dùng", http.StatusNotFound)
		return
	}

	// Map existing data to a temporary map for comparison
	currentData := docSnap.Data()

	// 4. Build update map & Check for changes
	updates := make(map[string]interface{})
	hasChanges := false

	// Helper logic to compare and add to update map
	compareAndAdd := func(field string, newValue interface{}) {

		// Compare logic
		if oldValue, ok := currentData[field]; ok && oldValue != newValue {
			hasChanges = true
			updates[field] = newValue
		}
	}

	compareAndAdd("name", req.Name)
	compareAndAdd("age", req.Age)
	compareAndAdd("address", req.Address)
	compareAndAdd("gender", req.Gender)
	compareAndAdd("phone_number", req.PhoneNumber)

	// 5. Duplicate Check
	if !hasChanges {
		sendJSONError(w, "The update is duplicated (Thông tin không thay đổi)", http.StatusConflict)
		return
	}

	// 6. Execute Update
	_, err = h.DB.Collection("users").Doc(userID).Set(ctx, updates, frMA)
	if err != nil {
		sendJSONError(w, "Lỗi cập nhật Firestore", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Cập nhật hồ sơ thành công",
	})
}
