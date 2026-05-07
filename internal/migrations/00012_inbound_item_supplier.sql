-- +goose Up
ALTER TABLE inbound_shipment_items
    ADD COLUMN supplier_name TEXT;

UPDATE inbound_shipment_items i
SET supplier_name = s.supplier_name
FROM inbound_shipments s
WHERE i.shipment_id = s.id
  AND i.supplier_name IS NULL;

ALTER TABLE inbound_shipment_items
    ALTER COLUMN supplier_name SET NOT NULL;

-- +goose Down
ALTER TABLE inbound_shipment_items
    DROP COLUMN IF EXISTS supplier_name;
