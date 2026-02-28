package db

import (
	"context"
	"log"
	"path/filepath"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// InitFirestore khởi tạo kết nối, trả về Client và Firebase App để dùng
func InitFirestore() (*firestore.Client, *firebase.App, error) {
	ctx := context.Background()
	url := "dbKey.json"

	// Đường dẫn tới file key.
	// Lưu ý: Đảm bảo file serviceAccountKey.json nằm cùng cấp với main.go khi chạy hoặc dùng đường dẫn tuyệt đối
	serviceAccountPath := url

	// Config đường dẫn tuyệt đối cho chắc ăn (fix lỗi không tìm thấy file khi chạy ở cmd khác)
	absPath, _ := filepath.Abs(serviceAccountPath)
	log.Printf("Đang load key tại: %s", absPath)

	opt := option.WithCredentialsFile(absPath)
	conf := &firebase.Config{ProjectID: "datahub-0912"}
	// Khởi tạo App
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		return nil, nil, err
	}

	// Lấy Firestore Client
	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, nil, err
	}

	return client, app, nil
}
