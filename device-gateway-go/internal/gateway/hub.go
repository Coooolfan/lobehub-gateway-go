package gateway

import (
	"errors"
	"sync"
	"time"
)

var errUserIDMismatch = errors.New("userId mismatch")

type pendingRequest struct {
	resolve func(rpcEnvelope)
	timer   *time.Timer
}

type hub struct {
	connections map[string]*connection
	pending     map[string]pendingRequest
	mu          sync.RWMutex
	userID      string
}

func newHub(userID string) *hub {
	return &hub{
		connections: map[string]*connection{},
		pending:     map[string]pendingRequest{},
		userID:      userID,
	}
}

func (h *hub) register(conn *connection) {
	h.mu.Lock()
	old := h.connections[conn.att.DeviceID]
	h.connections[conn.att.DeviceID] = conn
	h.mu.Unlock()

	if old != nil && old != conn {
		old.close(wsCloseNormal, "Replaced by new connection")
	}
}

func (h *hub) remove(conn *connection) {
	h.mu.Lock()
	if current := h.connections[conn.att.DeviceID]; current == conn {
		delete(h.connections, conn.att.DeviceID)
	}
	h.mu.Unlock()
	if conn.heartbeatTimer != nil {
		conn.heartbeatTimer.Stop()
	}
}

func (h *hub) markAuthenticated(conn *connection) {
	h.mu.Lock()
	conn.att.Authenticated = true
	conn.att.LastHeartbeat = time.Now().UnixMilli()
	h.mu.Unlock()
}

func (h *hub) recordHeartbeat(conn *connection) {
	h.mu.Lock()
	conn.att.LastHeartbeat = time.Now().UnixMilli()
	h.mu.Unlock()
}

func (h *hub) isAuthenticated(conn *connection) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return conn.att.Authenticated
}

func (h *hub) authenticatedConnections() []*connection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	connections := make([]*connection, 0, len(h.connections))
	for _, conn := range h.connections {
		if conn.att.Authenticated {
			connections = append(connections, conn)
		}
	}
	return connections
}

func (h *hub) devices() []DeviceAttachment {
	h.mu.RLock()
	defer h.mu.RUnlock()
	devices := make([]DeviceAttachment, 0, len(h.connections))
	for _, conn := range h.connections {
		if conn.att.Authenticated {
			devices = append(devices, conn.att)
		}
	}
	return devices
}

func (h *hub) target(deviceID string) *connection {
	connections := h.authenticatedConnections()
	if len(connections) == 0 {
		return nil
	}
	if deviceID == "" {
		return connections[0]
	}
	for _, conn := range connections {
		if conn.att.DeviceID == deviceID {
			return conn
		}
	}
	return nil
}

func (h *hub) setPending(key string, timeout time.Duration, resolve func(rpcEnvelope), onTimeout func()) {
	timer := time.AfterFunc(timeout, func() {
		h.mu.Lock()
		delete(h.pending, key)
		h.mu.Unlock()
		onTimeout()
	})
	h.mu.Lock()
	h.pending[key] = pendingRequest{resolve: resolve, timer: timer}
	h.mu.Unlock()
}

func (h *hub) resolvePending(msg rpcEnvelope) {
	key := msg.RequestID
	if key == "" {
		key = msg.OperationID
	}
	if key == "" {
		return
	}
	h.mu.Lock()
	pending, ok := h.pending[key]
	if ok {
		delete(h.pending, key)
	}
	h.mu.Unlock()
	if !ok {
		return
	}
	pending.timer.Stop()
	pending.resolve(msg)
}
