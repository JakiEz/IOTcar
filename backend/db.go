package main

import (
	"database/sql" // Go's generic SQL interface — works with any driver
	"encoding/json"
	"fmt"
	"math"

	// BLANK IMPORT: the "_" means "run this package's init() but don't give me a
	// name for it". The driver registers itself with database/sql under the
	// name "mysql" — that's all we need from it.
	_ "github.com/go-sql-driver/mysql"
)

// openDB connects to MySQL. The DSN (data source name) looks like:
//   user:password@tcp(127.0.0.1:3306)/cmt?parseTime=true
func openDB(dsn string) (*sql.DB, error) {
	// sql.Open does NOT connect — it only checks the DSN is well-formed.
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	// Ping actually opens a connection, so bad credentials fail here, at startup.
	// fmt.Errorf with %w "wraps" the original error so callers can still inspect it.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	db.SetMaxOpenConns(10) // *sql.DB is a connection POOL, not one connection
	return db, nil
}

// packStats derives pack-level numbers from the cells (mirrors packStats in the
// frontend's mockTruck.js). Uses NAMED RETURN VALUES: the results are declared
// in the signature, so a bare "return" sends back whatever they hold.
func packStats(cells []Cell) (totalPct, deltaMv int, maxTemp float64) {
	if len(cells) == 0 {
		return // all three stay at their zero values
	}
	sum := 0
	minV, maxV := cells[0].Voltage, cells[0].Voltage
	maxTemp = cells[0].TempC
	for _, c := range cells {
		sum += c.Pct
		if c.Voltage < minV {
			minV = c.Voltage
		}
		if c.Voltage > maxV {
			maxV = c.Voltage
		}
		if c.TempC > maxTemp {
			maxTemp = c.TempC
		}
	}
	totalPct = sum / len(cells)
	deltaMv = int(math.Round((maxV - minV) * 1000))
	return
}

// ---- WRITES (the simulator uses these; later Node-RED does this job) ----------

// Backtick strings are RAW string literals: they can span lines and contain
// quotes without escaping — ideal for SQL. The "?" are placeholders; the
// driver substitutes the args safely (no SQL injection possible).
const upsertStatusSQL = `
INSERT INTO truck_status
  (truck_id, online, last_seen, lat, lng, speed_kph, odometer_km, trip_km, motor_temp_c,
   pack_voltage, pack_current_a, total_pct, range_km, cells_json, delta_mv, max_temp_c)
VALUES (?, ?, NOW(), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  online = VALUES(online), last_seen = NOW(),
  lat = VALUES(lat), lng = VALUES(lng), speed_kph = VALUES(speed_kph),
  odometer_km = VALUES(odometer_km), trip_km = VALUES(trip_km), motor_temp_c = VALUES(motor_temp_c),
  pack_voltage = VALUES(pack_voltage), pack_current_a = VALUES(pack_current_a),
  total_pct = VALUES(total_pct), range_km = VALUES(range_km), cells_json = VALUES(cells_json),
  delta_mv = VALUES(delta_mv), max_temp_c = VALUES(max_temp_c)`

// upsertStatus writes one truck's live row (insert, or update if it exists).
func upsertStatus(db *sql.DB, t Truck) error {
	cellsJSON, err := json.Marshal(t.Battery.Cells) // struct → JSON bytes for the JSON column
	if err != nil {
		return err
	}
	totalPct, deltaMv, maxTemp := packStats(t.Battery.Cells)

	// db.Exec = run a statement that returns no rows. We ignore the result
	// (rows affected) with "_" and keep only the error.
	_, err = db.Exec(upsertStatusSQL,
		t.ID, t.Online, t.Position[0], t.Position[1], t.SpeedKph,
		t.Odometer.TotalKm, t.Odometer.TripKm, t.Energy.MotorTempC,
		t.Battery.PackVoltage, t.Battery.PackCurrentA, totalPct, t.Energy.RangeKm,
		cellsJSON, deltaMv, maxTemp)
	return err
}

// insertSnapshot copies every online truck's live row into the telemetry
// history table — the same job as the 5-second timer in the Node-RED flow.
const snapshotSQL = `
INSERT INTO telemetry
  (truck_id, ts, lat, lng, speed_kph, odometer_km, motor_temp_c,
   pack_voltage, pack_current_a, total_pct, cells_json, delta_mv, max_temp_c)
SELECT truck_id, NOW(3), lat, lng, speed_kph, odometer_km, motor_temp_c,
       pack_voltage, pack_current_a, total_pct, cells_json, delta_mv, max_temp_c
FROM truck_status
WHERE online = TRUE`

func insertSnapshot(db *sql.DB) error {
	_, err := db.Exec(snapshotSQL)
	return err
}

// ---- READS (what the API and WebSocket serve) ---------------------------------

// COALESCE(col, 0) turns SQL NULL into 0 so we can Scan straight into plain Go
// numbers. (Go also has sql.NullFloat64 etc. for when you must distinguish NULL,
// but COALESCE keeps this beginner-friendly.) CAST(... AS SIGNED) makes MySQL
// hand FLOAT columns to us as integers where our struct field is an int.
const fleetSQL = `
SELECT t.id, t.name, s.online,
       COALESCE(s.lat, 0), COALESCE(s.lng, 0),
       CAST(COALESCE(s.speed_kph, 0) AS SIGNED),
       COALESCE(s.odometer_km, 0), COALESCE(s.trip_km, 0), COALESCE(s.motor_temp_c, 0),
       COALESCE(s.pack_voltage, 0), COALESCE(s.pack_current_a, 0),
       CAST(COALESCE(s.range_km, 0) AS SIGNED),
       COALESCE(s.cells_json, '[]')
FROM trucks t
JOIN truck_status s ON s.truck_id = t.id
ORDER BY t.id`

const historySQL = `
SELECT COALESCE(lat, 0), COALESCE(lng, 0), COALESCE(total_pct, 0)
FROM telemetry
WHERE truck_id = ?
ORDER BY ts DESC
LIMIT 30`

// loadFleet builds the []Truck the frontend expects, from truck_status (live
// values) plus the last 30 telemetry rows (trail + battery history).
func loadFleet(db *sql.DB) ([]Truck, error) {
	rows, err := db.Query(fleetSQL) // db.Query = a statement that returns rows
	if err != nil {
		return nil, err
	}
	defer rows.Close() // always close rows — they hold a pooled connection

	var fleet []Truck
	for rows.Next() { // advance to the next row; false when done
		var t Truck
		var cellsRaw []byte
		// Scan copies the row's columns into the pointers, IN ORDER — the list
		// must match the SELECT column order exactly.
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Online,
			&t.Position[0], &t.Position[1], &t.SpeedKph,
			&t.Odometer.TotalKm, &t.Odometer.TripKm, &t.Energy.MotorTempC,
			&t.Battery.PackVoltage, &t.Battery.PackCurrentA, &t.Energy.RangeKm,
			&cellsRaw,
		); err != nil {
			return nil, err
		}
		// JSON bytes → []Cell. Guarantee a non-nil slice so JSON output is []
		// (not null) — the React battery panel indexes into it.
		if err := json.Unmarshal(cellsRaw, &t.Battery.Cells); err != nil || t.Battery.Cells == nil {
			t.Battery.Cells = []Cell{}
		}
		t.Energy.ConsumptionKwhPer100km = 112 // not stored in the DB yet
		fleet = append(fleet, t)
	}
	if err := rows.Err(); err != nil { // an error during iteration shows up here
		return nil, err
	}

	// Second pass: history per truck. (Can't run another query while still
	// iterating the rows above — that connection is busy.)
	for i := range fleet {
		if err := loadHistory(db, &fleet[i]); err != nil {
			return nil, err
		}
	}
	return fleet, nil
}

// loadHistory fills Trail and History from the newest 30 telemetry rows.
func loadHistory(db *sql.DB, t *Truck) error {
	rows, err := db.Query(historySQL, t.ID) // the arg fills the "?" placeholder
	if err != nil {
		return err
	}
	defer rows.Close()

	var trail [][2]float64
	var hist []int
	for rows.Next() {
		var lat, lng float64
		var pct int
		if err := rows.Scan(&lat, &lng, &pct); err != nil {
			return err
		}
		trail = append(trail, [2]float64{lat, lng})
		hist = append(hist, pct)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Rows arrived newest-first; reverse to chronological order for the chart.
	// This two-index loop is Go's idiomatic in-place reverse (no built-in).
	for i, j := 0, len(trail)-1; i < j; i, j = i+1, j-1 {
		trail[i], trail[j] = trail[j], trail[i]
		hist[i], hist[j] = hist[j], hist[i]
	}
	if trail == nil {
		trail = [][2]float64{t.Position} // at least the current point
	}
	if hist == nil {
		hist = []int{}
	}
	t.Trail, t.History = trail, hist
	return nil
}
