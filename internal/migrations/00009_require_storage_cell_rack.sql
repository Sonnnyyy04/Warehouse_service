-- +goose Up
INSERT INTO racks (code, name, zone, status)
VALUES ('RACK-DEFAULT', 'Стеллаж по умолчанию', 'default', 'active')
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    zone = EXCLUDED.zone,
    status = EXCLUDED.status;

UPDATE storage_cells
SET rack_id = (SELECT id FROM racks WHERE code = 'RACK-DEFAULT')
WHERE rack_id IS NULL;

INSERT INTO markers (marker_code, object_type, object_id)
SELECT 'MRK-RACK-' || LPAD(id::TEXT, 3, '0'), 'rack', id
FROM racks
WHERE code = 'RACK-DEFAULT'
ON CONFLICT (marker_code) DO NOTHING;

ALTER TABLE storage_cells
    ALTER COLUMN rack_id SET NOT NULL;

-- +goose Down
ALTER TABLE storage_cells
    ALTER COLUMN rack_id DROP NOT NULL;
