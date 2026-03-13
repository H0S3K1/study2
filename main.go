package main

import (
	"log"
	"net/http"
	"os"
	"study2/cmd/db"
	"study2/cmd/handler"
	"study2/cmd/middleware"
	"study2/cmd/repository"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Không tìm thấy file .env, load biến môi trường hệ thống")
	}

	// 1. Initialise singleton connections
	client, firebaseApp, err := db.InitFirestore()
	if err != nil {
		log.Fatalf("Lỗi kết nối Firestore: %v", err)
	}
	defer client.Close()
	db.InitRedis()

	// 2. Construct ProductRepository
	productRepo := &repository.ProductRepository{DB: client}

	// 3. Inject all dependencies into AppHandler
	app := &handler.AppHandler{
		DB:             client,
		App:            firebaseApp,
		FirebaseAPIKey: os.Getenv("FIREBASE_API_KEY"),
		Repo:           productRepo,
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux) // 3. Đăng ký routes

	log.Println("Server đang chạy tại cổng 8080...")
	loggedMux := middleware.LoggingMiddleware(mux)
	// 4. Mở port
	log.Fatal(http.ListenAndServe(":8080", loggedMux))
}

/// $(go env GOPATH)/bin/air ///
