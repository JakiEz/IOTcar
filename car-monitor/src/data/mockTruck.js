// Mock telemetry for one electric cement truck.
// This is the single source of truth for the dashboard's shape — when you wire
// up the ESP32 / MQTT feed later, produce an object in exactly this format and
// the whole UI updates for free.

const rand = (min, max) => Math.random() * (max - min) + min;

// Build a fresh truck snapshot. Pass options to distinguish trucks in a fleet.
// `cellCount` is flexible — pass whatever your battery pack reports and the
// cell grid adapts.
export function makeTruck({
  id = "CMT-01",
  name = "Cement Truck 01",
  position = [13.7563, 100.5018], // Bangkok — swap for your depot
  totalKm = 48213,
  online = true,
  cellCount = 12,
} = {}) {
  const cells = Array.from({ length: cellCount }, (_, i) => {
    const voltage = rand(3.2, 4.15); // per-cell voltage, typical Li-ion range
    return {
      id: i + 1,
      voltage: +voltage.toFixed(3),
      tempC: +rand(24, 41).toFixed(1),
      // charge % roughly mapped from voltage (3.0V empty → 4.2V full)
      pct: Math.round(((voltage - 3.0) / (4.2 - 3.0)) * 100),
    };
  });

  return {
    id,
    name,
    online,
    position,
    trail: [position],
    speedKph: 0,
    odometer: { totalKm, tripKm: 0 },
    energy: { motorTempC: 38, consumptionKwhPer100km: 112, rangeKm: 180 },
    battery: {
      packVoltage: +cells.reduce((s, c) => s + c.voltage, 0).toFixed(1),
      packCurrentA: +rand(-40, 120).toFixed(1), // + = discharge, - = regen/charge
      cells,
    },
    history: Array.from({ length: 30 }, () => Math.round(rand(70, 90))),
  };
}

// A small fleet of trucks with distinct ids, names, and depot positions.
// Replace with your real vehicle roster when the ESP32 feed is live.
export function makeFleet(cellCount = 12) {
  return [
    makeTruck({ id: "CMT-01", name: "Cement Truck 01", position: [13.7563, 100.5018], totalKm: 48213, cellCount }),
    makeTruck({ id: "CMT-02", name: "Cement Truck 02", position: [13.7469, 100.5349], totalKm: 61204, cellCount }),
    makeTruck({ id: "CMT-03", name: "Cement Truck 03", position: [13.7231, 100.5150], totalKm: 33987, cellCount }),
    makeTruck({ id: "CMT-04", name: "Cement Truck 04", position: [13.7650, 100.4890], totalKm: 12045, online: false, cellCount }),
  ];
}

// Derived pack stats used all over the UI. Kept in one place so "total %",
// min/max cell and cell delta are always computed the same way.
export function packStats(truck) {
  const cells = truck.battery.cells;
  // A truck with no cell data yet (e.g. never reported) must not crash the UI.
  if (!cells || cells.length === 0) {
    const none = { id: 0, voltage: 0 };
    return { totalPct: 0, minCell: none, maxCell: none, deltaMv: 0, maxTempC: 0 };
  }
  const pcts = cells.map((c) => c.pct);
  const volts = cells.map((c) => c.voltage);
  const temps = cells.map((c) => c.tempC);
  const totalPct = Math.round(pcts.reduce((a, b) => a + b, 0) / cells.length);
  const minCell = cells.reduce((m, c) => (c.voltage < m.voltage ? c : m), cells[0]);
  const maxCell = cells.reduce((m, c) => (c.voltage > m.voltage ? c : m), cells[0]);
  return {
    totalPct,
    minCell,
    maxCell,
    deltaMv: Math.round((Math.max(...volts) - Math.min(...volts)) * 1000), // cell spread in mV
    maxTempC: Math.max(...temps),
  };
}

// Advance the simulation one step — mimics a live feed pushing new samples.
// Returns a NEW object so React re-renders. Delete this once the real feed exists.
export function tick(truck) {
  if (!truck.online) return truck; // offline trucks report nothing new
  const speed = Math.max(0, truck.speedKph + rand(-8, 9));
  const stepKm = speed / 3600; // ~1s tick
  const drift = 0.0006 * (speed / 40 + 0.2);
  const position = [
    truck.position[0] + rand(-drift, drift),
    truck.position[1] + rand(-drift, drift),
  ];

  const cells = truck.battery.cells.map((c) => {
    const voltage = Math.min(4.2, Math.max(3.0, c.voltage + rand(-0.004, 0.001)));
    return {
      ...c,
      voltage: +voltage.toFixed(3),
      tempC: +Math.min(55, Math.max(20, c.tempC + rand(-0.4, 0.5))).toFixed(1),
      pct: Math.round(((voltage - 3.0) / (4.2 - 3.0)) * 100),
    };
  });

  const totalPct = Math.round(
    cells.reduce((s, c) => s + c.pct, 0) / cells.length
  );

  return {
    ...truck,
    speedKph: +speed.toFixed(0),
    position,
    trail: [...truck.trail.slice(-60), position],
    odometer: {
      totalKm: +(truck.odometer.totalKm + stepKm).toFixed(1),
      tripKm: +(truck.odometer.tripKm + stepKm).toFixed(1),
    },
    energy: {
      ...truck.energy,
      motorTempC: +Math.min(90, Math.max(25, truck.energy.motorTempC + rand(-0.5, 0.6))).toFixed(0),
      rangeKm: Math.round(totalPct * 2.1),
    },
    battery: {
      ...truck.battery,
      packVoltage: +cells.reduce((s, c) => s + c.voltage, 0).toFixed(1),
      packCurrentA: +rand(-40, 120).toFixed(1),
      cells,
    },
    history: [...truck.history.slice(-29), totalPct],
  };
}
