-- +goose Up
INSERT INTO racks (code, name, zone, status)
VALUES ('RACK-A-001', 'Стеллаж A-001', 'A', 'active')
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    zone = EXCLUDED.zone,
    status = EXCLUDED.status;

UPDATE storage_cells
SET rack_id = (SELECT id FROM racks WHERE code = 'RACK-A-001')
WHERE code = 'A-01-01'
  AND rack_id IS NULL;

UPDATE boxes
SET storage_cell_id = (SELECT id FROM storage_cells WHERE code = 'A-01-01')
WHERE code = 'BOX-001';

DELETE FROM markers
WHERE marker_code = 'MRK-PALLET-001';

INSERT INTO markers (marker_code, object_type, object_id)
SELECT 'MRK-RACK-001', 'rack', id
FROM racks
WHERE code = 'RACK-A-001'
ON CONFLICT (marker_code) DO UPDATE
SET object_type = EXCLUDED.object_type,
    object_id = EXCLUDED.object_id;

-- +goose Down
DELETE FROM markers
WHERE marker_code = 'MRK-RACK-001';

UPDATE storage_cells
SET rack_id = NULL
WHERE code = 'A-01-01'
  AND rack_id = (SELECT id FROM racks WHERE code = 'RACK-A-001');

DELETE FROM racks
WHERE code = 'RACK-A-001';
