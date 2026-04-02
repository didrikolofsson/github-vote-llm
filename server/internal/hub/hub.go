package hub

import "sync"

type EventType string

const (
	EventFeatureCreated EventType = "feature_created"
	EventFeatureUpdated EventType = "feature_updated"
)

type Hub interface {
	Subscribe(repoID int64) chan EventType
	Unsubscribe(repoID int64, ch chan EventType)
	Publish(repoID int64, event EventType)
}

type HubImpl struct {
	mu      sync.RWMutex
	clients map[int64]map[chan EventType]struct{}
}

func NewHub() Hub {
	return &HubImpl{
		clients: make(map[int64]map[chan EventType]struct{}),
	}
}

func (h *HubImpl) Subscribe(repoID int64) chan EventType {
	ch := make(chan EventType, 1)
	h.mu.Lock()
	if h.clients[repoID] == nil {
		h.clients[repoID] = make(map[chan EventType]struct{})
	}
	h.clients[repoID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *HubImpl) Unsubscribe(repoID int64, ch chan EventType) {
	h.mu.Lock()
	if m, ok := h.clients[repoID]; ok {
		delete(m, ch)
		if len(m) == 0 {
			delete(h.clients, repoID)
		}
	}
	h.mu.Unlock()
}

func (h *HubImpl) Publish(repoID int64, event EventType) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients[repoID] {
		select {
		case ch <- event:
		default:
		}
	}
}
