CREATE TABLE user_locations (
                                user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                location_id BIGINT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
                                role TEXT,
                                granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                PRIMARY KEY (user_id, location_id)
);

CREATE INDEX idx_user_locations_location_id
    ON user_locations(location_id);
