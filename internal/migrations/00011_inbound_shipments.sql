-- +goose Up
CREATE TABLE product_aliases (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    supplier_name TEXT NOT NULL,
    alias_type TEXT NOT NULL DEFAULT 'supplier_article',
    alias_value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (supplier_name, alias_type, alias_value)
);

CREATE INDEX idx_product_aliases_product_id ON product_aliases(product_id);

CREATE TABLE inbound_shipments (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    supplier_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE inbound_shipment_items (
    id BIGSERIAL PRIMARY KEY,
    shipment_id BIGINT NOT NULL REFERENCES inbound_shipments(id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES products(id) ON DELETE SET NULL,
    supplier_article TEXT NOT NULL,
    product_name TEXT NOT NULL,
    unit TEXT NOT NULL DEFAULT 'pcs',
    total_quantity INTEGER NOT NULL,
    boxes_count INTEGER NOT NULL,
    quantity_per_box INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'unresolved',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbound_shipment_items_shipment_id ON inbound_shipment_items(shipment_id);
CREATE INDEX idx_inbound_shipment_items_product_id ON inbound_shipment_items(product_id);

CREATE TABLE inbound_shipment_boxes (
    id BIGSERIAL PRIMARY KEY,
    shipment_item_id BIGINT NOT NULL REFERENCES inbound_shipment_items(id) ON DELETE CASCADE,
    box_id BIGINT REFERENCES boxes(id) ON DELETE SET NULL,
    batch_id BIGINT REFERENCES batches(id) ON DELETE SET NULL,
    planned_quantity INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'planned',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbound_shipment_boxes_item_id ON inbound_shipment_boxes(shipment_item_id);

-- +goose Down
DROP TABLE IF EXISTS inbound_shipment_boxes;
DROP TABLE IF EXISTS inbound_shipment_items;
DROP TABLE IF EXISTS inbound_shipments;
DROP TABLE IF EXISTS product_aliases;
