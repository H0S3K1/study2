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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("======================================")
		log.Printf("Yêu cầu: %s %s | Thời gian: %s", r.Method, r.URL.Path, time.Since(start))
		log.Printf("======================================")
	})
}
