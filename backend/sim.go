package main

import (
	"database/sql"
	"log"
	"time"
)

// runSimulator stands in for the real ESP32 → EMQX → Node-RED pipeline: it
// keeps a fake fleet in memory, ticks it every second, and WRITES each truck
// into MySQL exactly the way Node-RED will. Turn it off with SIMULATE=false
// once real data is flowing — nothing else in the backend changes.
func runSimulator(db *sql.DB, cellCount int) {
	sim := makeFleet(cellCount)

	// Seed: write every truck once, including offline ones, so each row has a
	// cells_json array from the start (the dashboard expects cells to exist).
	for _, t := range sim {
		if err := upsertStatus(db, t); err != nil {
			log.Println("simulator seed:", err)
		}
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		n := 0
		for range ticker.C {
			n++
			for i := range sim {
				if !sim[i].Online {
					continue // offline trucks don't report
				}
				tick(&sim[i])
				if err := upsertStatus(db, sim[i]); err != nil {
					log.Println("simulator upsert:", err)
				}
			}
			// Every 5th tick, snapshot live rows into the telemetry history table.
			if n%5 == 0 {
				if err := insertSnapshot(db); err != nil {
					log.Println("simulator snapshot:", err)
				}
			}
		}
	}()
}
