package handler

import (
	"net/http"
	"study2/cmd/middleware"
	"study2/cmd/repository"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
)

// AppHandler is the central struct holding all injected dependencies.
type AppHandler struct {
	DB             *firestore.Client
	App            *firebase.App
	FirebaseAPIKey string
	Repo           *repository.ProductRepository
	AuthRepo       *repository.AuthRepository
}

// Hàm này là "Sổ hộ khẩu" - Gom hết API vào đây
func (h *AppHandler) RegisterRoutes(mux *http.ServeMux) {
	// Dùng Firebase App đã được khởi tạo từ main tiêm vào
	authMid := middleware.AuthMiddleware(h.App)
	protected := func(pettern string, handlefunc http.HandlerFunc) {
		mux.Handle(pettern, authMid(handlefunc))
	}
	// Public Group
	mux.HandleFunc("POST /login", h.LoginHandler)
	mux.HandleFunc("POST /login/google", h.GoogleLoginHandler)
	mux.HandleFunc("POST /login/facebook", h.FacebookLoginHandler)
	mux.HandleFunc("POST /login/apple", h.AppleLoginHandler)
	mux.HandleFunc("POST /refresh", h.Refresh)
	mux.HandleFunc("POST /register", h.RegisterHandler)
	mux.HandleFunc("GET /products", h.GetProductHandler)
	mux.HandleFunc("GET /products/{id}", h.GetProductByIDHandler)
	mux.HandleFunc("POST /search", h.SeachingProd)
	mux.HandleFunc("POST /products/filter", h.GetProductsByFilter)

	// Protected Group
	protected("GET /profile", h.GetProfileHandler)
	protected("POST /profile/update", h.UpdateProfileHandler)
	protected("POST /logout", h.LogoutHandler)
	protected("POST /cart/add", h.AddToCart)
	protected("POST /cart/remove", h.RemoveFromCart)
	protected("POST /cart/update", h.UpdateCart)
	protected("GET /cart", h.GetCart)
}
