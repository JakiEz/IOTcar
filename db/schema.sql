-- ============================================================================
--  Electric Cement Truck Monitor — MySQL schema
--  Flow: ESP32 → EMQX (MQTT) → Node-RED → MySQL → React dashboard
--
--  Design notes:
--   * truck_status = ONE row per truck, the live snapshot the dashboard reads.
--     Every MQTT message UPSERTs this row (always instant, no history scan).
--   * telemetry    = downsampled history (a periodic snapshot of truck_status,
--     ~1 row/truck/5s) for charts and trend queries. Cells stored as a JSON
--     array per sample, plus delta_mv / max_temp_c precomputed for fast health
--     queries. No per-cell table — that would balloon disk for little benefit.
--   * alerts       = discrete events (cell delta high, over-temp, offline).
-- ============================================================================

CREATE DATABASE IF NOT EXISTS cmt
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE cmt;

-- ---- Registry: static per-truck info -------------------------------------
CREATE TABLE IF NOT EXISTS trucks (
  id          VARCHAR(16)  NOT NULL,          -- 'CMT-01'
  name        VARCHAR(64)  NOT NULL,
  vin         VARCHAR(32)  NULL,
  cell_count  SMALLINT     NOT NULL DEFAULT 12,
  created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

-- ---- Live snapshot: what the dashboard queries (one row per truck) --------
CREATE TABLE IF NOT EXISTS truck_status (
  truck_id       VARCHAR(16) NOT NULL,
  online         BOOLEAN     NOT NULL DEFAULT FALSE,
  last_seen      TIMESTAMP   NULL,
  -- position / drivetrain (from cmt/{id}/telemetry)
  lat            DOUBLE      NULL,
  lng            DOUBLE      NULL,
  speed_kph      FLOAT       NULL,
  odometer_km    DOUBLE      NULL,
  trip_km        DOUBLE      NULL,
  motor_temp_c   FLOAT       NULL,
  -- battery pack (from cmt/{id}/battery)
  pack_voltage   FLOAT       NULL,
  pack_current_a FLOAT       NULL,
  total_pct      TINYINT     NULL,
  range_km       SMALLINT    NULL,
  cells_json     JSON        NULL,            -- latest per-cell array
  delta_mv       SMALLINT    NULL,            -- highest-lowest cell spread (mV)
  max_temp_c     FLOAT       NULL,            -- hottest cell
  updated_at     TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
                                       ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (truck_id),
  CONSTRAINT fk_status_truck FOREIGN KEY (truck_id)
    REFERENCES trucks(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- ---- History: downsampled time-series for charts --------------------------
CREATE TABLE IF NOT EXISTS telemetry (
  id             BIGINT       NOT NULL AUTO_INCREMENT,
  truck_id       VARCHAR(16)  NOT NULL,
  ts             TIMESTAMP(3) NOT NULL,
  lat            DOUBLE       NULL,
  lng            DOUBLE       NULL,
  speed_kph      FLOAT        NULL,
  odometer_km    DOUBLE       NULL,
  motor_temp_c   FLOAT        NULL,
  pack_voltage   FLOAT        NULL,
  pack_current_a FLOAT        NULL,
  total_pct      TINYINT      NULL,
  cells_json     JSON         NULL,           -- cell snapshot at this ts
  delta_mv       SMALLINT     NULL,
  max_temp_c     FLOAT        NULL,
  PRIMARY KEY (id),
  KEY idx_truck_ts (truck_id, ts)
) ENGINE=InnoDB;

-- ---- Alerts: discrete events ---------------------------------------------
CREATE TABLE IF NOT EXISTS alerts (
  id           BIGINT       NOT NULL AUTO_INCREMENT,
  truck_id     VARCHAR(16)  NOT NULL,
  ts           TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  type         VARCHAR(32)  NOT NULL,         -- 'cell_delta_high','over_temp','offline','low_soc'
  severity     ENUM('info','warning','critical') NOT NULL DEFAULT 'info',
  message      VARCHAR(255) NULL,
  resolved_at  TIMESTAMP    NULL,
  PRIMARY KEY (id),
  KEY idx_truck_ts (truck_id, ts),
  KEY idx_open (truck_id, resolved_at)
) ENGINE=InnoDB;

-- ---- Rejects: messages the Node-RED guard refused to store ----------------
-- Deliberately has NO foreign key on truck_id: an unknown truck_id is itself
-- one of the reasons a message gets rejected, so the row must still be storable.
CREATE TABLE IF NOT EXISTS rejects (
  id        BIGINT       NOT NULL AUTO_INCREMENT,
  ts        TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  truck_id  VARCHAR(16)  NULL,
  topic     VARCHAR(128) NULL,
  reason    VARCHAR(255) NOT NULL,
  payload   JSON         NULL,            -- the raw message, for debugging
  PRIMARY KEY (id),
  KEY idx_ts (ts),
  KEY idx_truck_ts (truck_id, ts)
) ENGINE=InnoDB;

-- ---- Seed the fleet (matches the frontend mock roster) --------------------
INSERT INTO trucks (id, name, cell_count) VALUES
  ('CMT-01', 'Cement Truck 01', 12),
  ('CMT-02', 'Cement Truck 02', 12),
  ('CMT-03', 'Cement Truck 03', 12),
  ('CMT-04', 'Cement Truck 04', 12)
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- Ensure a status row exists for each truck (starts offline).
INSERT INTO truck_status (truck_id, online) VALUES
  ('CMT-01', FALSE), ('CMT-02', FALSE), ('CMT-03', FALSE), ('CMT-04', FALSE)
ON DUPLICATE KEY UPDATE truck_id = truck_id;
