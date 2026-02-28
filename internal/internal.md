## models

# internal/models/product.go

type Product struct {
ID string `json:"id"`
Brand string `json:"brand"`
CPU string `json:"cpu"`
Name string `json:"name"`
Price float64 `json:"price"`
Type string `json:"type"`
}

type PingType struct {
ID string `json:"id" firestore:"-"`
Type string `json:"type" firestore:"type,omitempty"`
Brand string `json:"brand" firestore:"brand,omitempty"`
Name string `json:"name" firestore:"name,omitempty"`
CPU string `json:"cpu" firestore:"cpu,omitempty"`
//Images []string `json:"images" firestore:"images,omitempty"`
//Price int `json:"price" firestore:"price,omitempty"`
//Storage string `json:"storage" firestore:"storage,omitempty"`
//Thumbnail string `json:"thumbnail" firestore:"thumbnail,omitempty"`
}

type Require struct {
Field string
Value string
}

type ProductFilter struct {
Brands []string `json:"brands"`
CPUs []string `json:"cpus"`
MinPrice int `json:"min_price"`
MaxPrice int `json:"max_price"`
Type []string `json:"type"`
// Thêm các trường khác nếu cần
}

# internal/models/auth.go

type LoginRequest struct {
Email string `json:"email" validate:"required,email"`
Password string `json:"password" validate:"required,min=6"`
}

type LoginResponse struct {
IDToken string `json:"idToken"`
Email string `json:"email"`
RefreshToken string `json:"refreshToken"`
ExpiresIn string `json:"expiresIn"`
LocalID string `json:"localId"`
}

type RegisterRequest struct {
Email string `json:"email" validate:"required,email"`
Password string `json:"password" validate:"required,min=6"`
}

type RegisterResponse struct {
IDToken string `json:"idToken"`
Email string `json:"email"`
RefreshToken string `json:"refreshToken"`
ExpiresIn string `json:"expiresIn"`
LocalID string `json:"localId"`
}

type EditProfileRequest struct {
Password string `json:"password" validate:"omitempty,min=6"`
Email string `json:"email" validate:"omitempty,email"`
}

type UpdateProfileRequest struct {
Name string `json:"name,omitempty"`
Age int `json:"age,omitempty" validate:"omitempty,min=0,max=150"`
Address string `json:"address,omitempty"`
Gender string `json:"gender,omitempty" validate:"omitempty,oneof=male female other"`
PhoneNumber string `json:"phone_number,omitempty" validate:"omitempty,numeric,min=9,max=15"`
}

# internal/models/user.go

type User struct {
Email string `json:"email" firestore:"email"`
Role string `json:"role" firestore:"role"`
Name string `json:"name,omitempty" firestore:"name,omitempty"`
Age int `json:"age,omitempty" firestore:"age,omitempty"`
Address string `json:"address,omitempty" firestore:"address,omitempty"`
Balance int `json:"balance,omitempty" firestore:"balance,omitempty"`
Gender string `json:"gender,omitempty" firestore:"gender,omitempty"`
PhoneNumber string `json:"phone_number,omitempty" firestore:"phone_number,omitempty"`
}

## middleware

# internal/middleware/auth.go

package middleware

import (
"context"
"log"
"net/http"
"strings"
"time"

    firebase "firebase.google.com/go/v4"

)

// respondJSONError format lại lỗi thành JSON để api trả về chuẩn chỉnh
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(statusCode)
w.Write([]byte(`{"error": "` + message + `"}`))
}

// AuthMiddleware kiểm tra Token Firebase từ Header
func AuthMiddleware(app *firebase.App) func(http.Handler) http.Handler {
return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 1. Kiểm tra request Authorization Header
authHeader := r.Header.Get("Authorization")
if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
respondJSONError(w, "Thiếu hoặc sai format token (cần Bearer token)", http.StatusUnauthorized)
return
}

    		// 2. Tách lấy token từ Bearer string
    		idToken := strings.TrimPrefix(authHeader, "Bearer ")

    		// 3. Khởi tạo Auth Client từ Firebase App
    		client, err := app.Auth(context.Background())
    		if err != nil {
    			respondJSONError(w, "Lỗi kết nối Firebase Auth Client", http.StatusInternalServerError)
    			return
    		}

    		// 4. Xác thực token (kiểm tra hạn, tính toàn vẹn)
    		token, err := client.VerifyIDToken(context.Background(), idToken)
    		if err != nil {
    			respondJSONError(w, "Token không hợp lệ hoặc đã hết hạn", http.StatusUnauthorized)
    			return
    		}

    		// 5. Giải mã thành công -> Gán User ID vào (r.Context)
    		ctx := context.WithValue(r.Context(), "user_id", token.UID)

    		// 6. Cho phép gọi tiếp Handler tiếp theo (controller)
    		next.ServeHTTP(w, r.WithContext(ctx))
    	})
    }

}
func LoggingMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r \*http.Request) {
start := time.Now()
next.ServeHTTP(w, r)
log.Printf("======================================")
log.Printf("Yêu cầu: %s %s | Thời gian: %s", r.Method, r.URL.Path, time.Since(start))
log.Printf("======================================")
})
}

## handler

# internal/handler/auth.go

package handler

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
"os"
"study2/internal/model"
"time"
)

// helper function gửi json lỗi
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(statusCode)
json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// LoginHandler xử lý việc đăng nhập của user thông qua Firebase REST API
func (h *AppHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")

    // 1. Đọc request body
    var req model.LoginRequest
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
    firebaseAPIKey := os.Getenv("FIREBASE_API_KEY")
    firebaseURL := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", firebaseAPIKey)

    payload := map[string]interface{}{
    	"email":             req.Email,
    	"password":          req.Password,
    	"returnSecureToken": true, // Quan trọng: Yêu cầu Firebase trả về JWT Token thật
    }
    payloadBytes, _ := json.Marshal(payload)

    // 4. Gửi HTTP POST Request sang Google
    resp, err := http.Post(firebaseURL, "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
    	sendJSONError(w, "Không kết nối được với Firebase Authentication", http.StatusInternalServerError)
    	return
    }
    defer resp.Body.Close()

    // 5. Kiểm tra nếu Firebase báo lỗi (sai password, sai email, v.v)
    if resp.StatusCode != http.StatusOK {
    	sendJSONError(w, "Sai email hoặc mật khẩu", http.StatusUnauthorized)
    	return
    }

    // 6. Request thành công -> Parse kết quả và trả thẳng Token cho Client
    var loginResp model.LoginResponse
    if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
    	sendJSONError(w, "Lỗi đọc Token từ Firebase", http.StatusInternalServerError)
    	return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(loginResp)

}

// RegisterHandler xử lý việc đăng ký user mới thông qua Firebase REST API
func (h *AppHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")

    // 1. Đọc request body
    var req model.RegisterRequest
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
    firebaseAPIKey := os.Getenv("FIREBASE_API_KEY")
    firebaseURL := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signUp?key=%s", firebaseAPIKey)

    payload := map[string]interface{}{
    	"email":             req.Email,
    	"password":          req.Password,
    	"returnSecureToken": true, // Yêu cầu trả về token luôn sau khi đăng ký thành công
    }
    payloadBytes, _ := json.Marshal(payload)

    // 4. Gửi HTTP POST Request sang Google
    resp, err := http.Post(firebaseURL, "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
    	sendJSONError(w, "Không kết nối được với Firebase Authentication", http.StatusInternalServerError)
    	return
    }
    defer resp.Body.Close()

    // 5. Kiểm tra nếu Firebase báo lỗi (ví dụ: email đã tồn tại)
    if resp.StatusCode != http.StatusOK {
    	sendJSONError(w, "Đăng ký thất bại (có thể email đã được sử dụng)", http.StatusBadRequest)
    	return
    }

    // 6. Request thành công -> Parse kết quả trả về
    var regResp model.RegisterResponse
    if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
    	sendJSONError(w, "Đăng ký thành công nhưng gặp lỗi khi đọc thông tin phản hồi", http.StatusInternalServerError)
    	return
    }

    // 7. Lưu user profile (Filing Cabinet) vào Firestore!
    // Dùng LocalID (chính là UID của Firebase Auth) làm Document ID
    _, err = h.DB.Collection("users").Doc(regResp.LocalID).Set(r.Context(), map[string]interface{}{
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
    var req model.EditProfileRequest
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
    firebaseAPIKey := os.Getenv("FIREBASE_API_KEY")
    firebaseURL := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:update?key=%s", firebaseAPIKey)

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

    payloadBytes, _ := json.Marshal(payload)

    // 5. Gửi HTTP POST Request sang Google
    resp, err := http.Post(firebaseURL, "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
    	sendJSONError(w, "Không kết nối được với Firebase Authentication", http.StatusInternalServerError)
    	return
    }
    defer resp.Body.Close()

    // 6. Kiểm tra nếu Firebase báo lỗi
    if resp.StatusCode != http.StatusOK {
    	sendJSONError(w, "Thay đổi thất bại (token có thể đã hết hạn hoặc email đã được dùng)", http.StatusBadRequest)
    	return
    }

    // 7. Request thành công -> Parse kết quả và trả token mới về (vì đổi email/pass xong token có thể được cấp lại)
    var updateResp model.RegisterResponse // Dùng chung struct với register cho kết quả update
    if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
    	sendJSONError(w, "Cập nhật thành công nhưng gặp lỗi đọc phản hồi", http.StatusInternalServerError)
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

    // 3. Map dữ liệu trả về từ Firestore vào struct model.User
    var user model.User
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
    var req model.UpdateProfileRequest
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
    	oldValue, exists := currentData[field]

    	// Compare logic
    	isDifferent := false
    	if exists {
    		// Because Firestore saves numbers as float64 locally sometimes or int64, when we decode it, it gets messy. We force strings and generic format checks:
    		isDifferent = fmt.Sprintf("%v", oldValue) != fmt.Sprintf("%v", newValue)
    	} else {
    		isDifferent = true
    	}

    	// Only add if it's different AND the user actually provided a non-empty newValue
    	if isDifferent && newValue != "" && newValue != 0 {
    		updates[field] = newValue
    		hasChanges = true
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

# internal/handler/routes.go

package handler

import (
"net/http"
"study2/internal/middleware"

    "cloud.google.com/go/firestore"
    firebase "firebase.google.com/go/v4"

)

// Struct trung tâm giữ Client DB và Firebase App
type AppHandler struct {
DB *firestore.Client
App *firebase.App
}
var frMA = firestore.MergeAll

// Hàm này là "Sổ hộ khẩu" - Gom hết API vào đây
func (h *AppHandler) RegisterRoutes(mux *http.ServeMux) {
// Dùng Firebase App đã được khởi tạo từ main tiêm vào
authMid := middleware.AuthMiddleware(h.App)
protected := func(pettern string, handlefunc http.HandlerFunc) {
mux.Handle(pettern, authMid(handlefunc))
}
// Public Group
mux.HandleFunc("POST /login", h.LoginHandler)
mux.HandleFunc("POST /register", h.RegisterHandler)
mux.HandleFunc("GET /products", h.GetProductHandler)

    // Protected Group
    protected("POST /products/filter", h.GetProductByFillter)
    protected("PUT /profile", h.EditProfileHandler)
    protected("GET /profile", h.GetProfileHandler)
    protected("POST /profile/update", h.UpdateProfileHandler)

}
