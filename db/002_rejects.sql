-- Migration: add the rejects table to an EXISTING cmt database.
-- (schema.sql only auto-runs on a brand-new/empty MySQL volume, so databases
--  created before this table existed need it applied by hand.)
--
-- Apply with:
--   cd D:\projects\IOTcar
--   docker compose exec -T mysql sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD"' < db\002_rejects.sql

USE cmt;

CREATE TABLE IF NOT EXISTS rejects (
  id        BIGINT       NOT NULL AUTO_INCREMENT,
  ts        TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  truck_id  VARCHAR(16)  NULL,
  topic     VARCHAR(128) NULL,
  reason    VARCHAR(255) NOT NULL,
  payload   JSON         NULL,
  PRIMARY KEY (id),
  KEY idx_ts (ts),
  KEY idx_truck_ts (truck_id, ts)
) ENGINE=InnoDB;
