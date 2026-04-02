package hub

import "sync"

const (
	EventFeatureCreated = "feature_created"
	EventFeatureUpdated = "feature_updated"
)

type Hub interface {
	Subscribe(repoID int64) chan string
	Unsubscribe(repoID int64, ch chan string)
	Publish(repoID int64, event string)
}

type HubImpl struct {
	mu      sync.RWMutex
	clients map[int64]map[chan string]struct{}
}

func NewHub() Hub {
	return &HubImpl{
		clients: make(map[int64]map[chan string]struct{}),
	}
}

func (h *HubImpl) Subscribe(repoID int64) chan string {
	ch := make(chan string, 1)
	h.mu.Lock()
	if h.clients[repoID] == nil {
		h.clients[repoID] = make(map[chan string]struct{})
	}
	h.clients[repoID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *HubImpl) Unsubscribe(repoID int64, ch chan string) {
	h.mu.Lock()
	if m, ok := h.clients[repoID]; ok {
		delete(m, ch)
		if len(m) == 0 {
			delete(h.clients, repoID)
		}
	}
	h.mu.Unlock()
}

func (h *HubImpl) Publish(repoID int64, event string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients[repoID] {
		select {
		case ch <- event:
		default:
		}
	}
}
