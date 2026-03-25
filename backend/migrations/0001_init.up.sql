-- Включаем расширения (нужны UUID и PostGIS)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- users
CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                     email VARCHAR(255) NOT NULL UNIQUE,
                                     password_hash VARCHAR(255) NOT NULL,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                     updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- refresh_tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
                                              id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                              user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                              token_hash VARCHAR(255) NOT NULL UNIQUE,
                                              expires_at TIMESTAMPTZ NOT NULL,
                                              revoked_at TIMESTAMPTZ NULL,
                                              created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens(user_id);

-- tasks
CREATE TABLE IF NOT EXISTS tasks (
                                     id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                     user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                     title VARCHAR(200) NOT NULL,
                                     address_text VARCHAR(400) NOT NULL,
                                     latitude NUMERIC(9,6) NOT NULL,
                                     longitude NUMERIC(9,6) NOT NULL,
                                     duration_min INT NOT NULL,
                                     window_start TIME NULL,
                                     window_end TIME NULL,
                                     sort_index INT NOT NULL DEFAULT 0,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                     updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                     is_deleted BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS tasks_user_id_idx ON tasks(user_id);
CREATE INDEX IF NOT EXISTS tasks_user_sort_idx ON tasks(user_id, sort_index);

-- routes
CREATE TABLE IF NOT EXISTS routes (
                                      id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                      user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                      status VARCHAR(30) NOT NULL,   -- draft|optimized|failed
                                      source VARCHAR(30) NOT NULL,   -- manual|optimized
                                      algorithm VARCHAR(50) NULL,
                                      started_at TIMESTAMPTZ NULL,
                                      finished_at TIMESTAMPTZ NULL,
                                      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS routes_user_id_idx ON routes(user_id);

-- route_stats (1:1)
CREATE TABLE IF NOT EXISTS route_stats (
                                           route_id UUID PRIMARY KEY REFERENCES routes(id) ON DELETE CASCADE,
                                           total_distance_m INT NULL,
                                           total_travel_sec INT NULL,
                                           total_service_sec INT NULL,
                                           total_wait_sec INT NULL,
                                           computed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- route_stops
CREATE TABLE IF NOT EXISTS route_stops (
                                           id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                           route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
                                           task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
                                           position INT NOT NULL,
                                           travel_from_prev_sec INT NULL,
                                           arrive_time TIMESTAMPTZ NULL,
                                           service_start_time TIMESTAMPTZ NULL,
                                           service_end_time TIMESTAMPTZ NULL,
                                           wait_sec INT NULL
);
CREATE INDEX IF NOT EXISTS route_stops_route_id_idx ON route_stops(route_id);
CREATE UNIQUE INDEX IF NOT EXISTS route_stops_route_position_uq ON route_stops(route_id, position);

-- route_geometry (1:1)
CREATE TABLE IF NOT EXISTS route_geometry (
                                              route_id UUID PRIMARY KEY REFERENCES routes(id) ON DELETE CASCADE,
                                              polyline TEXT NOT NULL,
                                              bbox_json JSONB NULL,
                                              geojson_json JSONB NULL,
                                              updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- api_distance_cache
CREATE TABLE IF NOT EXISTS api_distance_cache (
                                                  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                                  provider VARCHAR(30) NOT NULL,
                                                  origin geometry(Point, 4326) NOT NULL,
  dest geometry(Point, 4326) NOT NULL,
  distance_m INT NOT NULL,
  duration_sec INT NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  request_hash VARCHAR(64) NOT NULL UNIQUE
);
CREATE INDEX IF NOT EXISTS api_distance_cache_origin_gix ON api_distance_cache USING GIST(origin);
CREATE INDEX IF NOT EXISTS api_distance_cache_dest_gix ON api_distance_cache USING GIST(dest);
