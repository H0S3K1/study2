package main

import (
	"log"
	"net/http"
	"study2/internal/db"
	"study2/internal/handler" // Import cái package handler ông vừa tạo
	"study2/internal/middleware"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Không tìm thấy file .env, load biến môi trường hệ thống")
	}

	// 1. Khởi tạo kết nối duy nhất (Singleton)
	client, firebaseApp, err := db.InitFirestore()
	if err != nil {
		log.Fatalf("Lỗi kết nối Firestore: %v", err)
	}
	defer client.Close()

	// 2. "Tiêm" (Inject) cái client và app vào AppHandler
	app := &handler.AppHandler{
		DB:  client,
		App: firebaseApp,
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux) // 3. Đăng ký routes

	log.Println("Server đang chạy tại cổng 8080...")
	loggedMux := middleware.LoggingMiddleware(mux)
	// 4. Mở port
	log.Fatal(http.ListenAndServe(":8080", loggedMux))
}

/// $(go env GOPATH)/bin/air ///
