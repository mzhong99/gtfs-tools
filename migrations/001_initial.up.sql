CREATE TABLE IF NOT EXISTS feed_version (
    feed_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    imported_at TIMESTAMPTZ NOT NULL,
    source_url TEXT,
    source_sha256 TEXT
);

CREATE TABLE IF NOT EXISTS agency (
    agency_id TEXT PRIMARY KEY,
    agency_name TEXT,
    agency_url TEXT,
    agency_timezone TEXT,
    agency_lang TEXT,
    agency_phone TEXT,

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id)
);

CREATE TABLE IF NOT EXISTS routes (
    route_id TEXT PRIMARY KEY,
    agency_id TEXT,
    route_short_name TEXT,
    route_long_name TEXT,
    route_desc TEXT,
    route_type SMALLINT NOT NULL,
    route_url TEXT,
    route_color CHAR(6),
    route_text_color CHAR(6),
    route_sort_order INTEGER,

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id)
);

CREATE TABLE IF NOT EXISTS trips (
    trip_id TEXT PRIMARY KEY,
    route_id TEXT,
    service_id TEXT,
    trip_headsign TEXT,
    direction_id SMALLINT,
    shape_id TEXT,

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id),

    rt_trip_id TEXT GENERATED ALWAYS AS (
        substring(trip_id FROM '[0-9]+_[^_]+$')
    ) STORED,

    UNIQUE(rt_trip_id, service_id)
);

CREATE INDEX idx_trips_rt_trip_id ON trips(rt_trip_id);

CREATE TABLE IF NOT EXISTS stops (
    stop_id TEXT PRIMARY KEY,
    stop_name TEXT,
    stop_lat DOUBLE PRECISION,
    stop_lon DOUBLE PRECISION,
    location_type SMALLINT,
    parent_station TEXT,

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id)
);

CREATE TABLE IF NOT EXISTS stop_times (
    stop_id TEXT,
    trip_id TEXT,
    arrival_time TEXT,
    departure_time TEXT,
    stop_sequence INTEGER,
    PRIMARY KEY (trip_id, stop_sequence),

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id)
);

CREATE TABLE IF NOT EXISTS calendar (
    service_id TEXT PRIMARY KEY,
    monday BOOLEAN,
    tuesday BOOLEAN,
    wednesday BOOLEAN,
    thursday BOOLEAN,
    friday BOOLEAN,
    saturday BOOLEAN,
    sunday BOOLEAN,
    start_date DATE,
    end_date DATE,

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id)
);

CREATE TABLE IF NOT EXISTS calendar_dates (
    service_id TEXT,
    date DATE,
    exception_type SMALLINT,
    PRIMARY KEY (service_id, date),

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id)
);

CREATE TABLE IF NOT EXISTS shapes (
    shape_id TEXT,
    shape_pt_lat DOUBLE PRECISION,
    shape_pt_lon DOUBLE PRECISION,
    shape_pt_sequence INTEGER,
    shape_dist_traveled DOUBLE PRECISION,
    PRIMARY KEY (shape_id, shape_pt_sequence),

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id)
);

CREATE TABLE IF NOT EXISTS transfers (
    from_stop_id TEXT,
    to_stop_id TEXT,
    transfer_type SMALLINT,
    min_transfer_time INTEGER,
    PRIMARY KEY (from_stop_id, to_stop_id),

    feed_id BIGINT,
    foreign key (feed_id) references feed_version(feed_id)
);