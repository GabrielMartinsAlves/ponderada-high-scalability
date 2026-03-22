CREATE TABLE IF NOT EXISTS telemetry_data (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    recorded_at TIMESTAMP NOT NULL,
    sensor_type VARCHAR(100) NOT NULL,
    reading_nature VARCHAR(50) NOT NULL, -- discrete or analog
    value FLOAT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
