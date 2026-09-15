package main

import (
	"database/sql"
	"log"
	"sync" // provides the mutex (lock) types
	"time"
)

// ---- Store: the shared fleet, guarded by a lock ----------------------------
// The Store bundles the data (fleet) together with the lock (mu) that protects
// it. This is the idiomatic Go pattern: keep the mutex right next to the thing
// it guards.
type Store struct {
	mu    sync.RWMutex // RW = separate locks for readers vs writers
	fleet []Truck
}

// newStore returns a *Store (POINTER) so every caller shares the one instance
// and therefore the one mutex — a mutex must never be copied.
func newStore() *Store {
	return &Store{fleet: []Truck{}}
}

// set replaces the whole fleet (called by the DB reader each second).
func (s *Store) set(fleet []Truck) {
	s.mu.Lock() // WRITE lock — exclusive
	defer s.mu.Unlock()
	s.fleet = fleet
}

// forEachReadLocked runs fn with the fleet while holding the READ lock, so a
// concurrent set() can't swap the data mid-use.
func (s *Store) forEachReadLocked(fn func(fleet []Truck)) {
	s.mu.RLock() // READ lock — shared; many readers at once, blocked by a writer
	defer s.mu.RUnlock()
	fn(s.fleet)
}

// runReader is the background goroutine that pulls the fleet from MySQL every
// second, stores it, and hands it to onUpdate (the WebSocket broadcast).
// Its only source of truth is the database — it doesn't care who wrote the
// rows (our simulator today, Node-RED tomorrow).
func runReader(db *sql.DB, store *Store, onUpdate func(fleet []Truck)) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fleet, err := loadFleet(db)
			if err != nil {
				log.Println("reader:", err)
				continue // skip this tick, try again next second
			}
			store.set(fleet)
			onUpdate(fleet)
		}
	}()
}

// ---- tick: mutate one truck (used by the simulator) --------------------------
// Takes *Truck (a pointer) so changes stick to the caller's truck.
func tick(t *Truck) {
	if !t.Online {
		return
	}

	speed := t.SpeedKph + int(randRange(-8, 9))
	if speed < 0 {
		speed = 0
	}
	if speed > 90 {
		speed = 90 // a loaded cement truck is not doing 113 km/h
	}
	t.SpeedKph = speed

	// nudge the GPS position and extend the trail (capped at 60 points)
	drift := 0.0006
	t.Position[0] += randRange(-drift, drift)
	t.Position[1] += randRange(-drift, drift)
	t.Trail = append(t.Trail, t.Position)
	if len(t.Trail) > 60 {
		t.Trail = t.Trail[len(t.Trail)-60:]
	}

	// odometer: distance this ~1s tick
	stepKm := float64(speed) / 3600.0
	t.Odometer.TotalKm += stepKm
	t.Odometer.TripKm += stepKm

	// Cell temp drifts TOWARD a speed-dependent target (mean reversion) and is
	// clamped. A plain "+= random" walk has no bound and runs away — an earlier
	// version of this drifted to 750°C over a few hours.
	tempTarget := 30.0 + float64(speed)/6.0

	// walk each cell, drift its voltage/temp, recompute charge %
	total := 0
	packV := 0.0
	for i := range t.Battery.Cells {
		c := &t.Battery.Cells[i]
		c.Voltage += randRange(-0.004, 0.001)
		if c.Voltage > 4.2 {
			c.Voltage = 4.2
		}
		if c.Voltage < 3.2 {
			c.Voltage = 4.1 // "recharged at the depot" — keeps the sim cycling
		}
		c.Voltage = round(c.Voltage, 3)
		c.TempC = round(c.TempC+(tempTarget-c.TempC)*0.05+randRange(-0.3, 0.3), 1)
		if c.TempC > 55 {
			c.TempC = 55
		}
		if c.TempC < 20 {
			c.TempC = 20
		}
		c.Pct = int((c.Voltage - 3.0) / (4.2 - 3.0) * 100)
		total += c.Pct
		packV += c.Voltage
	}

	totalPct := total / len(t.Battery.Cells)
	t.Energy.RangeKm = int(float64(totalPct) * 2.1)
	t.Energy.MotorTempC = round(t.Energy.MotorTempC+randRange(-0.5, 0.6), 0)
	t.Battery.PackVoltage = round(packV, 1)
	t.Battery.PackCurrentA = round(randRange(-40, 120), 1)

	t.History = append(t.History, totalPct)
	if len(t.History) > 30 {
		t.History = t.History[len(t.History)-30:]
	}
}
