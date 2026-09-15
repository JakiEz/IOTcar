package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv" // loads KEY=VALUE lines from a .env file into env vars
)

// envOr reads an environment variable, falling back to def if it's unset.
// Reading config from the environment (not hardcoding) is what makes the same
// binary work on your laptop, in Docker, and in the cloud.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// withCORS is MIDDLEWARE: wraps a handler so every response carries CORS headers.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Load backend/.env if it exists. We ignore the error on purpose: in Docker
	// or the cloud the variables come from the environment, not a file.
	_ = godotenv.Load()

	port := envOr("PORT", "8080")
	simulate := envOr("SIMULATE", "true") == "true"
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is not set — copy .env.example to .env and fill in your MySQL password")
	}

	db, err := openDB(dsn)
	if err != nil {
		log.Fatal(err) // can't run without the database
	}
	defer db.Close()
	log.Println("connected to MySQL")

	store := newStore()
	hub := newHub()

	// Writer side: the simulator stands in for Node-RED until real data flows.
	if simulate {
		runSimulator(db, 12)
		log.Println("simulator ON (set SIMULATE=false once Node-RED writes real data)")
	}

	// Load once now so the very first request isn't empty, then keep reading.
	if fleet, err := loadFleet(db); err == nil {
		store.set(fleet)
	} else {
		log.Println("initial load:", err)
	}
	runReader(db, store, hub.broadcast)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/trucks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		store.forEachReadLocked(func(fleet []Truck) {
			if err := json.NewEncoder(w).Encode(fleet); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.serveWS(store, w, r)
	})

	log.Printf("Server running on http://localhost:%s  (REST: /api/trucks, WS: /ws)", port)
	log.Fatal(http.ListenAndServe(":"+port, withCORS(mux)))
}
