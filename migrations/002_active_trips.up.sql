CREATE TABLE IF NOT EXISTS feed_snapshots (
    snapshot_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Table is appended data from GTFS-RT feed, which is a snapshot of all active trips at a given time.
-- This is used to determine where each train is at a given time, and what trips are associated with it.
--
-- Data is appended with an insertion time so that we can prune old data after a certain amount of time
-- has passed (e.g. 24 hours).
--
-- Ah... the start date, time, and trip id shouldn't ever change for a given train. So that would allow
-- us to uniquely identify a train by the combination of those three fields.
CREATE TABLE IF NOT EXISTS trip_update_events (
    -- Used to create 1:N relationship with trip_update_stop_time_events
    trip_id TEXT,           -- What trip you're taking
    start_date TEXT,        -- Date at which trip was started
    start_time TEXT,        -- Time of day at which trip was started
    direction_id SMALLINT,
    snapshot_id BIGINT,

    insertion_time TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT snapshot_id
        FOREIGN KEY (snapshot_id) REFERENCES feed_snapshots(snapshot_id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS trip_update_stop_time_events (
    stop_id TEXT,
    arrival_time BIGINT,
    departure_time BIGINT,
    snapshot_id BIGINT,

    CONSTRAINT snapshot_id
        FOREIGN KEY (snapshot_id) REFERENCES feed_snapshots(snapshot_id)
        ON DELETE CASCADE
);

-- Need to decide how best to model trains and associated trips - I want to be able to easily query
-- to ask where every train is at a given time, and what trips are associated with it.

-- Need:
-- * Representation per train
-- * Representation of one route
-- * Representation of one station

-- Train view
CREATE OR REPLACE VIEW view_train_trips AS
WITH latest AS (
    SELECT DISTINCT ON (trip_id, start_date)
        trip_id,
        start_date,
        start_time,
        direction_id,
        snapshot_id,
        insertion_time
    FROM trip_update_events
    ORDER BY
        trip_id,
        start_date,
        insertion_time DESC,
        snapshot_id DESC
)
SELECT
    latest.trip_id,
    latest.start_date,
    latest.direction_id as rt_direction_id,
    trips.route_id,
    trips.trip_headsign
FROM latest
LEFT JOIN trips ON trips.rt_trip_id = latest.trip_id
