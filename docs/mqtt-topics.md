# MQTT Topics — Electric Cement Truck Monitor

**Scheme:** `cmt/{truckId}/{channel}` — e.g. `cmt/CMT-01/battery`.

Flow: **ESP32 → EMQX → Node-RED → MySQL → React**. The ESP32 on each truck
**publishes**; the Node-RED backend **subscribes** with wildcards and writes to
MySQL.

## Topic table

| Topic | Direction | QoS | Retained | Purpose |
|---|---|---|---|---|
| `cmt/{id}/status` | ESP32 → backend | 1 | ✅ | Online/offline. Also the **Last Will (LWT)** — see below. |
| `cmt/{id}/telemetry` | ESP32 → backend | 0 | ❌ | Position, speed, odometer, motor temp. High rate, loss-tolerant. |
| `cmt/{id}/battery` | ESP32 → backend | 0 | ✅ | Pack totals + all cell voltages/temps. Retained → dashboard shows last-known on reconnect. |
| `cmt/{id}/alert` | ESP32 → backend | 1 | ❌ | Fault events raised on the truck (optional; backend also derives alerts). |
| `cmt/{id}/cmd/{action}` | backend → ESP32 | 1 | ❌ | Downlink commands (e.g. `reset-trip`). Future. |

**Backend subscriptions (wildcards):**
`cmt/+/status`, `cmt/+/telemetry`, `cmt/+/battery`, `cmt/+/alert`

## Last Will & Testament (LWT)

Configure the ESP32's MQTT connection so the broker publishes this automatically
if the truck drops off (power loss, out of coverage):

- **LWT topic:** `cmt/{id}/status`
- **LWT payload:** `{"online": false}`
- **LWT QoS / retain:** 1 / true

On connect, the truck publishes `{"online": true}` (retained) to the same topic.
This means "offline" is detected by the broker even when the ESP32 can't send
anything itself.

## Payloads

All payloads are JSON, flat where possible (easy for the ESP32 to build). `ts`
is ISO-8601 UTC; if omitted the backend stamps arrival time.

### `cmt/{id}/status`
```json
{ "online": true }
```

### `cmt/{id}/telemetry`
```json
{
  "ts": "2026-09-12T08:15:03Z",
  "lat": 13.7563,
  "lng": 100.5018,
  "speedKph": 42,
  "odometerKm": 48213.2,
  "tripKm": 12.4,
  "motorTempC": 38
}
```

### `cmt/{id}/battery`
```json
{
  "ts": "2026-09-12T08:15:03Z",
  "packVoltage": 46.8,
  "packCurrentA": 80.4,
  "totalPct": 57,
  "rangeKm": 120,
  "cells": [
    { "id": 1, "voltage": 3.457, "tempC": 39.1, "pct": 42 }
  ]
}
```
> The backend computes `delta_mv` (max−min cell, in mV) and `max_temp_c` from
> the `cells` array — the ESP32 doesn't need to send them.

### `cmt/{id}/alert` (optional; backend also derives these)
```json
{
  "ts": "2026-09-12T08:15:03Z",
  "type": "over_temp",
  "severity": "warning",
  "message": "Cell 7 at 52.3°C"
}
```

## Backend-derived alerts

The Node-RED battery handler raises alerts without the truck needing to
(thresholds are starting points — tune to your pack):

| Condition | type | severity |
|---|---|---|
| `delta_mv > 150` | `cell_delta_high` | warning |
| `max_temp_c > 50` | `over_temp` | critical |
| `total_pct < 15` | `low_soc` | warning |
| LWT fires (`online:false`) | `offline` | info |

## How messages map to the database

| Topic | Action |
|---|---|
| `status` | UPSERT `truck_status` (online, last_seen) |
| `telemetry` | UPSERT `truck_status` (position, speed, odometer, motor) |
| `battery` | UPSERT `truck_status` (pack, cells_json, total, delta_mv, max_temp_c) + raise alerts |
| _(timer, every 5 s)_ | Snapshot online rows of `truck_status` → INSERT `telemetry` (history) |
| `alert` | INSERT `alerts` |
