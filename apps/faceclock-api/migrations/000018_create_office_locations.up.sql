CREATE TABLE IF NOT EXISTS office_locations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(100) NOT NULL,
    latitude double precision NOT NULL,
    longitude double precision NOT NULL,
    radius_meter integer NOT NULL DEFAULT 100,
    is_active boolean NOT NULL DEFAULT true,
    address text,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT office_locations_lat_chk CHECK (latitude >= -90 AND latitude <= 90),
    CONSTRAINT office_locations_lon_chk CHECK (longitude >= -180 AND longitude <= 180),
    CONSTRAINT office_locations_radius_chk CHECK (radius_meter >= 10 AND radius_meter <= 50000)
);

CREATE UNIQUE INDEX IF NOT EXISTS office_locations_name_uidx ON office_locations (lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS office_locations_active_idx ON office_locations (is_active) WHERE is_active = true AND deleted_at IS NULL;

-- Seed default location per Plan/05-Fase4.md § 2.2
INSERT INTO office_locations (name, latitude, longitude, radius_meter, is_active, address)
VALUES ('Kantor Pusat', -6.2088, 106.8456, 100, true, 'Jakarta Pusat')
ON CONFLICT DO NOTHING;
