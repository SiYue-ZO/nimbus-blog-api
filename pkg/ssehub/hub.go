// Package ssehub SSE Hub（内存级连接管理与事件推送）。
package ssehub

import "sync"

const _defaultBufferSize = 16

// Event SSE 事件。
type Event struct {
	Name string
	Data []byte
}

// Hub SSE Hub。
type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[chan Event]struct{}
}

// New 创建 Hub。
func New() *Hub {
	return &Hub{clients: make(map[int64]map[chan Event]struct{})}
}

// Subscribe 订阅指定用户的事件通道。
func (h *Hub) Subscribe(userID int64) chan Event {
	ch := make(chan Event, _defaultBufferSize)
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[chan Event]struct{})
	}
	h.clients[userID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe 取消订阅并关闭事件通道。
func (h *Hub) Unsubscribe(userID int64, ch chan Event) {
	h.mu.Lock()
	if conns, ok := h.clients[userID]; ok {
		delete(conns, ch)
		if len(conns) == 0 {
			delete(h.clients, userID)
		}
	}
	h.mu.Unlock()
	close(ch)
}

// Publish 向指定用户推送事件。
func (h *Hub) Publish(userID int64, event Event) {
	h.mu.RLock()
	conns := h.clients[userID]
	h.mu.RUnlock()

	for ch := range conns {
		select {
		case ch <- event:
		default:
		}
	}
}

// Shutdown 关闭 Hub 并清理所有订阅。
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for uid, conns := range h.clients {
		for ch := range conns {
			close(ch)
		}
		delete(h.clients, uid)
	}
}
