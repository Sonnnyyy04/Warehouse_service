-- +goose Up

-- demo user
INSERT INTO users (email, full_name, role, password_hash)
VALUES (
           'demo.worker@example.com',
           'Demo Worker',
           'worker',
           '$2a$10$demo.hash.for.seed.only'
       );

-- storage cell
INSERT INTO storage_cells (code, name, zone, status)
VALUES (
           'A-01-01',
           'Ячейка A-01-01',
           'A',
           'active'
       );

-- box in storage cell
INSERT INTO boxes (code, status, storage_cell_id)
VALUES (
           'BOX-001',
           'active',
           (SELECT id FROM storage_cells WHERE code = 'A-01-01')
       );

-- product
INSERT INTO products (sku, name, unit)
VALUES (
           'SKU-0001',
           'Молоток 500г',
           'pcs'
       );

-- batch inside box
INSERT INTO batches (code, product_id, quantity, status, box_id, storage_cell_id)
VALUES (
           'BATCH-2026-0001',
           (SELECT id FROM products WHERE sku = 'SKU-0001'),
           25,
           'active',
           (SELECT id FROM boxes WHERE code = 'BOX-001'),
           NULL
       );

-- markers
INSERT INTO markers (marker_code, object_type, object_id)
VALUES
    (
        'MRK-CELL-001',
        'storage_cell',
        (SELECT id FROM storage_cells WHERE code = 'A-01-01')
    ),
    (
        'MRK-BOX-001',
        'box',
        (SELECT id FROM boxes WHERE code = 'BOX-001')
    ),
    (
        'MRK-PRODUCT-001',
        'product',
        (SELECT id FROM products WHERE sku = 'SKU-0001')
    ),
    (
        'MRK-BATCH-001',
        'batch',
        (SELECT id FROM batches WHERE code = 'BATCH-2026-0001')
    );

-- optional demo operation history
INSERT INTO operation_history (object_type, object_id, operation_type, user_id, details)
VALUES (
           'box',
           (SELECT id FROM boxes WHERE code = 'BOX-001'),
           'seed_created',
           (SELECT id FROM users WHERE email = 'demo.worker@example.com'),
           '{"source":"seed","comment":"demo box created for manual API testing"}'::jsonb
       );

-- +goose Down

DELETE FROM operation_history
WHERE operation_type = 'seed_created';

DELETE FROM markers
WHERE marker_code IN (
                      'MRK-CELL-001',
                      'MRK-BOX-001',
                      'MRK-PRODUCT-001',
                      'MRK-BATCH-001'
    );

DELETE FROM batches
WHERE code = 'BATCH-2026-0001';

DELETE FROM products
WHERE sku = 'SKU-0001';

DELETE FROM boxes
WHERE code = 'BOX-001';

DELETE FROM storage_cells
WHERE code = 'A-01-01';

DELETE FROM users
WHERE email = 'demo.worker@example.com';
