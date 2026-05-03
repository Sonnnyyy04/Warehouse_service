-- +goose Up
ALTER TYPE object_type ADD VALUE IF NOT EXISTS 'rack';

CREATE TABLE racks (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    zone TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE storage_cells
    ADD COLUMN rack_id BIGINT REFERENCES racks(id) ON DELETE SET NULL;

CREATE INDEX idx_storage_cells_rack_id ON storage_cells(rack_id);

-- +goose Down
DROP INDEX IF EXISTS idx_storage_cells_rack_id;

ALTER TABLE storage_cells
    DROP COLUMN IF EXISTS rack_id;

DROP TABLE IF EXISTS racks;
