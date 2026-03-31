-- +goose Up
CREATE TYPE object_type AS ENUM (
'storage_cell',
'pallet',
'box',
'product',
'batch'
);

CREATE TABLE users (
id BIGSERIAL PRIMARY KEY,
email TEXT NOT NULL UNIQUE,
full_name TEXT NOT NULL,
role TEXT NOT NULL DEFAULT 'worker',
password_hash TEXT NOT NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE storage_cells (
id BIGSERIAL PRIMARY KEY,
code TEXT NOT NULL UNIQUE,
name TEXT NOT NULL,
zone TEXT,
status TEXT NOT NULL DEFAULT 'active',
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE pallets (
id BIGSERIAL PRIMARY KEY,
code TEXT NOT NULL UNIQUE,
status TEXT NOT NULL DEFAULT 'active',
storage_cell_id BIGINT REFERENCES storage_cells(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE boxes (
id BIGSERIAL PRIMARY KEY,
code TEXT NOT NULL UNIQUE,
status TEXT NOT NULL DEFAULT 'active',
pallet_id BIGINT REFERENCES pallets(id) ON DELETE SET NULL,
storage_cell_id BIGINT REFERENCES storage_cells(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE products (
id BIGSERIAL PRIMARY KEY,
sku TEXT NOT NULL UNIQUE,
name TEXT NOT NULL,
unit TEXT NOT NULL DEFAULT 'pcs',
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE batches (
id BIGSERIAL PRIMARY KEY,
code TEXT NOT NULL UNIQUE,
product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
quantity INTEGER NOT NULL DEFAULT 0,
status TEXT NOT NULL DEFAULT 'active',
box_id BIGINT REFERENCES boxes(id) ON DELETE SET NULL,
pallet_id BIGINT REFERENCES pallets(id) ON DELETE SET NULL,
storage_cell_id BIGINT REFERENCES storage_cells(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE markers (
id BIGSERIAL PRIMARY KEY,
marker_code TEXT NOT NULL UNIQUE,
object_type object_type NOT NULL,
object_id BIGINT NOT NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_markers_marker_code ON markers(marker_code);
CREATE INDEX idx_markers_object_type_object_id ON markers(object_type, object_id);

CREATE TABLE scan_events (
id BIGSERIAL PRIMARY KEY,
marker_code TEXT NOT NULL,
user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
device_info TEXT,
success BOOLEAN NOT NULL DEFAULT TRUE,
scanned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE operation_history (
id BIGSERIAL PRIMARY KEY,
object_type object_type NOT NULL,
object_id BIGINT NOT NULL,
operation_type TEXT NOT NULL,
user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
details JSONB,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS operation_history;
DROP TABLE IF EXISTS scan_events;
DROP TABLE IF EXISTS markers;
DROP TABLE IF EXISTS batches;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS boxes;
DROP TABLE IF EXISTS pallets;
DROP TABLE IF EXISTS storage_cells;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS object_type;