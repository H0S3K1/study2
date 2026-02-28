package handler

import (
	"fmt"
	"net/http"
	"os"
	"study2/internal/middleware"
	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
)

// Struct trung tâm giữ Client DB và Firebase App
type AppHandler struct {
	DB  *firestore.Client
	App *firebase.App
}

var frMA = firestore.MergeAll
var firebaseAPIKey = os.Getenv("FIREBASE_API_KEY")
var firebaseURL = fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", firebaseAPIKey)

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
