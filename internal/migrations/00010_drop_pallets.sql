-- +goose Up
DELETE FROM markers
WHERE object_type::text = 'pallet'
   OR marker_code LIKE 'MRK-PALLET-%';

DELETE FROM operation_history
WHERE object_type::text = 'pallet';

DO $$
BEGIN
    IF to_regclass('public.pallets') IS NOT NULL
       AND EXISTS (
           SELECT 1
           FROM information_schema.columns
           WHERE table_schema = 'public'
             AND table_name = 'boxes'
             AND column_name = 'pallet_id'
       )
    THEN
        UPDATE boxes b
        SET storage_cell_id = COALESCE(b.storage_cell_id, p.storage_cell_id)
        FROM pallets p
        WHERE b.pallet_id = p.id;
    END IF;
END $$;

DO $$
BEGIN
    IF to_regclass('public.pallets') IS NOT NULL
       AND EXISTS (
           SELECT 1
           FROM information_schema.columns
           WHERE table_schema = 'public'
             AND table_name = 'batches'
             AND column_name = 'pallet_id'
       )
    THEN
        UPDATE batches b
        SET storage_cell_id = COALESCE(b.storage_cell_id, p.storage_cell_id)
        FROM pallets p
        WHERE b.pallet_id = p.id
          AND b.box_id IS NULL;
    END IF;
END $$;

ALTER TABLE boxes
    DROP COLUMN IF EXISTS pallet_id;

ALTER TABLE batches
    DROP COLUMN IF EXISTS pallet_id;

DROP TABLE IF EXISTS pallets;

DROP TYPE IF EXISTS object_type_without_pallet;

CREATE TYPE object_type_without_pallet AS ENUM (
    'storage_cell',
    'box',
    'product',
    'batch',
    'rack'
);

ALTER TABLE markers
    ALTER COLUMN object_type TYPE object_type_without_pallet
    USING object_type::text::object_type_without_pallet;

ALTER TABLE operation_history
    ALTER COLUMN object_type TYPE object_type_without_pallet
    USING object_type::text::object_type_without_pallet;

DROP TYPE object_type;

ALTER TYPE object_type_without_pallet RENAME TO object_type;

-- +goose Down
DROP TYPE IF EXISTS object_type_with_pallet;

CREATE TYPE object_type_with_pallet AS ENUM (
    'storage_cell',
    'box',
    'product',
    'batch',
    'rack',
    'pallet'
);

ALTER TABLE markers
    ALTER COLUMN object_type TYPE object_type_with_pallet
    USING object_type::text::object_type_with_pallet;

ALTER TABLE operation_history
    ALTER COLUMN object_type TYPE object_type_with_pallet
    USING object_type::text::object_type_with_pallet;

DROP TYPE object_type;

ALTER TYPE object_type_with_pallet RENAME TO object_type;

CREATE TABLE IF NOT EXISTS pallets (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active',
    storage_cell_id BIGINT REFERENCES storage_cells(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE boxes
    ADD COLUMN IF NOT EXISTS pallet_id BIGINT REFERENCES pallets(id) ON DELETE SET NULL;

ALTER TABLE batches
    ADD COLUMN IF NOT EXISTS pallet_id BIGINT REFERENCES pallets(id) ON DELETE SET NULL;
