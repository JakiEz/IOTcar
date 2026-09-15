package main

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket" // the dependency we just `go get`-ed
)

// upgrader turns an incoming HTTP request into a WebSocket connection.
// CheckOrigin returning true allows connections from any origin — fine for
// local dev (your React app runs on a different port). Tighten this in prod.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ---- Hub: the set of connected browsers -------------------------------------
// clients is a "set" done the Go way: a map whose values are bool. Go has no
// built-in set type, so map[T]bool is the idiom — presence of the key = member.
type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func newHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

// add registers a new browser and immediately sends it the current fleet, so it
// isn't blank until the next tick. Done under the lock so this write can't race
// with a broadcast write on the same connection.
func (h *Hub) add(conn *websocket.Conn, initial []Truck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	conn.WriteJSON(initial) // WriteJSON = encode value to JSON and send it
}

// remove drops a browser (called when it disconnects).
func (h *Hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn) // delete removes a key from a map
	conn.Close()
}

// broadcast sends the fleet to every connected browser. Its signature matches
// the onUpdate callback the store's ticker expects, so we can hand it straight
// to startTicker. If a write fails, that client is gone — drop it.
func (h *Hub) broadcast(fleet []Truck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteJSON(fleet); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

// serveWS is the HTTP handler for /ws. It upgrades the request, registers the
// browser, then blocks reading until the connection closes.
func (h *Hub) serveWS(store *Store, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // upgrade failed (not a websocket request); nothing to do
	}

	// Send the current fleet snapshot, captured under the store's read lock.
	store.forEachReadLocked(func(fleet []Truck) {
		h.add(conn, fleet)
	})

	// Keep the connection alive. We don't expect messages FROM the browser, but
	// we must keep reading so the library can detect a close/ping. When the
	// browser disconnects, ReadMessage returns an error and we clean up.
	defer h.remove(conn)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
