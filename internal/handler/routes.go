package handler

import (
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
var FirebaseAPIKey = os.Getenv("FIREBASE_API_KEY")

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
	mux.HandleFunc("GET /products/{id}", h.GetProductByIDHandler)
	mux.HandleFunc("POST /search", h.SeachingProd)
	mux.HandleFunc("POST /products/filter", h.GetProductsByFilter)

	// Protected Group
	protected("PUT /profile", h.EditProfileHandler)
	protected("GET /profile", h.GetProfileHandler)
	protected("POST /profile/update", h.UpdateProfileHandler)
	protected("POST /logout", h.LogoutHandler)
	protected("POST /cart/add", h.AddToCart)
	protected("POST /cart/remove", h.RemoveFromCart)
	protected("POST /cart/update", h.UpdateCart)
	protected("GET /cart", h.GetCart)
}
