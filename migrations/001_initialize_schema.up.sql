CREATE TABLE gauges (
                       id TEXT PRIMARY KEY NOT NULL,
                       value double precision NOT NULL
);
CREATE TABLE counters (
                       id TEXT PRIMARY KEY NOT NULL,
                       value bigint NOT NULL
);
