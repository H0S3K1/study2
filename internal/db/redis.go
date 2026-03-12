
package db // Thả vào chung folder db của ông

import (
	"sync"
	"time"
)

// Cấu trúc 1 item lưu trong RAM
type CacheItem struct {
	Data      interface{}
	ExpiresAt int64
}

// Cấu trúc bộ nhớ Cache
type LocalCache struct {
	mu    sync.RWMutex
	items map[string]CacheItem
}

// Khởi tạo đồ nhà trồng (Dùng chung cho toàn app)
var MyCache = &LocalCache{
	items: make(map[string]CacheItem),
}

// Ghi vào RAM
func (c *LocalCache) Set(key string, data interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = CacheItem{
		Data:      data,
		ExpiresAt: time.Now().Add(duration).UnixNano(),
	}
}

// Đọc từ RAM ra
func (c *LocalCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// Check xem hết hạn chưa (Ví dụ: lưu 5 phút)
	if time.Now().UnixNano() > item.ExpiresAt {
		// Ở hệ thống lớn người ta sẽ tự động xóa, mình đơn giản thì cứ báo false
		return nil, false 
	}
	return item.Data, true
}