package main

// This file is in "package main" too — Go files in the same folder share a
// package and can use each other's types directly, with no import between them.

import (
	"math"      // math.Round, math.Pow — for rounding floats
	"math/rand" // Go's random-number generator, for the fake data.
)

// round trims a float to `places` decimals. Go has no built-in "toFixed" that
// returns a number, so this is the standard trick: scale up, round, scale down.
func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places)) // float64(...) converts int→float
	return math.Round(v*p) / p
}

// ---- The data model, as Go structs -----------------------------------------
// A struct is a fixed shape: a named set of typed fields. Think of it as the
// TypeScript "interface" you'd write for mockTruck.js, but enforced at compile
// time. Field name comes first, its TYPE second, then an optional `json:"..."`
// TAG telling the JSON encoder what key to use (Go needs Capital field names,
// JSON wants camelCase — the tag bridges the two).

type Cell struct {
	ID      int     `json:"id"`
	Voltage float64 `json:"voltage"` // float64 = Go's standard decimal number
	TempC   float64 `json:"tempC"`
	Pct     int     `json:"pct"`
}

type Battery struct {
	PackVoltage  float64 `json:"packVoltage"`
	PackCurrentA float64 `json:"packCurrentA"`
	Cells        []Cell  `json:"cells"` // []Cell = a slice (list) of Cell
}

type Odometer struct {
	TotalKm float64 `json:"totalKm"`
	TripKm  float64 `json:"tripKm"`
}

type Energy struct {
	MotorTempC             float64 `json:"motorTempC"`
	ConsumptionKwhPer100km float64 `json:"consumptionKwhPer100km"`
	RangeKm                int     `json:"rangeKm"`
}

type Truck struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Online   bool         `json:"online"`
	Position [2]float64   `json:"position"` // [2]float64 = fixed array of 2 (lat, lng)
	Trail    [][2]float64 `json:"trail"`    // slice of [lat,lng] points
	SpeedKph int          `json:"speedKph"`
	Odometer Odometer     `json:"odometer"` // structs can nest inside structs
	Energy   Energy       `json:"energy"`
	Battery  Battery      `json:"battery"`
	History  []int        `json:"history"`
}

// ---- Building fake data -----------------------------------------------------

// randRange returns a random float between min and max.
// The "(min, max float64) float64" part means: takes two float64s, RETURNS a
// float64. Return type goes at the end of the signature.
func randRange(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// makeTruck builds one fake truck. Parameters are typed; it returns a Truck.
func makeTruck(id, name string, pos [2]float64, totalKm float64, online bool, cellCount int) Truck {
	// Build the cell slice. "make([]Cell, cellCount)" pre-allocates a slice of
	// cellCount empty Cells — like "new Array(n)" in JS.
	cells := make([]Cell, cellCount)
	for i := 0; i < cellCount; i++ {
		v := randRange(3.2, 4.15)
		cells[i] = Cell{
			ID:      i + 1,
			Voltage: round(v, 3),
			TempC:   round(randRange(24, 41), 1),
			Pct:     int((v - 3.0) / (4.2 - 3.0) * 100), // int(...) converts float→int
		}
	}

	// Sum the cell voltages for the pack total. ":=" declares AND assigns a new
	// variable, letting Go infer its type (packV becomes a float64 here).
	packV := 0.0
	for _, c := range cells { // range gives (index, value); "_" ignores the index
		packV += c.Voltage
	}

	// Construct and return the Truck. Naming each field (ID:, Name:, ...) is the
	// clear, idiomatic way to build a struct.
	return Truck{
		ID:       id,
		Name:     name,
		Online:   online,
		Position: pos,
		Trail:    [][2]float64{pos},
		SpeedKph: 0,
		Odometer: Odometer{TotalKm: totalKm, TripKm: 0},
		Energy:   Energy{MotorTempC: 38, ConsumptionKwhPer100km: 112, RangeKm: 180},
		Battery: Battery{
			PackVoltage:  round(packV, 1),
			PackCurrentA: round(randRange(-40, 120), 1),
			Cells:        cells,
		},
		History: []int{},
	}
}

// makeFleet returns the starting fleet as a slice of Truck.
func makeFleet(cellCount int) []Truck {
	return []Truck{
		makeTruck("CMT-01", "Cement Truck 01", [2]float64{13.7563, 100.5018}, 48213, true, cellCount),
		makeTruck("CMT-02", "Cement Truck 02", [2]float64{13.7469, 100.5349}, 61204, true, cellCount),
		makeTruck("CMT-03", "Cement Truck 03", [2]float64{13.7231, 100.5150}, 33987, true, cellCount),
		makeTruck("CMT-04", "Cement Truck 04", [2]float64{13.7650, 100.4890}, 12045, false, cellCount),
	}
}
