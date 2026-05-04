# AGENTS.md

## Purpose

This repository contains the backend of a diploma project: a warehouse mobile application backend for identification and accounting of warehouse objects.

The backend is already implemented as an MVP and is the source of truth for:
- domain model,
- API contracts,
- current business flows,
- storage rules,
- object types,
- demo scenario used by the mobile client.

Your task in this repository is usually **not** to reinvent the backend, but to:
- understand how it works,
- preserve the current architecture,
- make careful incremental changes when explicitly requested,cvjn
- keep the API stable for the mobile client.

---

## Project summary

The system operates on warehouse entities:ghb
- storage cells,
- pallets,
- boxes,
- products,
- batches,
- markers (QR / barcode / marker code),
- scan events,
- operation history,
- users.

Main user-facing flow:
1. a worker scans a marker;
2. backend resolves the marker to a business object;
3. backend returns an object card;
4. backend optionally records a scan event;
5. worker can perform warehouse actions such as moving a box to another storage cell;
6. backend records the business operation in history.

At the current stage, backend MVP is complete enough to support a mobile client.

---

## Tech stack

- Go 1.24
- PostgreSQL
- Docker / Docker Compose
- goose migrations
- pgx/v5 + pgxpool
- config via:
    - github.com/joho/godotenv
    - github.com/caarlos0/env/v11

Important:
- the project runs in Docker;
- migrations run in Docker;
- do not suggest moving the migration workflow to a purely local setup;
- do not require Go 1.25 just for migrations.

---

## Non-negotiable constraints

You must preserve these decisions unless the user explicitly asks to change them:

1. Do not change the project structure.
2. Do not propose microservices.
3. Do not collapse the logic into one giant SQL query for by-marker resolution.
4. Do not remove interfaces between layers.
5. Do not move orchestration out of the service layer.
6. Do not replace `models` with another folder such as `object`.
7. Do not suggest installing goose locally as the main workflow.
8. Do not casually redesign the domain model when only small feature work is needed.

---

## Fixed project structure

```text
Warehouse_service/
├─ cmd/
│  └─ api/
│     └─ main.go
├─ internal/
│  ├─ config/
│  │  └─ config.go
│  ├─ handler/
│  ├─ migrations/
│  ├─ models/
│  ├─ repository/
│  └─ service/
├─ pkg/
├─ .env
├─ docker-compose.yml
├─ Dockerfile
├─ go.mod
└─ go.sum
```

## Architectural rules

Architecture is clean-ish and intentionally simple:

handler -> service -> repository
models are separate
interfaces are declared in the consumer layer

That means:

handler declares service/use-case interfaces it depends on;
service declares repository interfaces it depends on;
repository contains concrete DB implementation.
## Responsibilities
Handler layer

Responsible for:

HTTP parsing,
validation of request shape,
calling use cases,
mapping errors to HTTP responses,
returning JSON.

Handler layer must not contain orchestration or DB logic.

Service layer

Responsible for:

use-case orchestration,
business rules,
calling multiple repositories,
deciding business-level errors,
composing response models.

Service layer is where orchestration must live.

Repository layer

Responsible for:

SQL,
persistence,
DB access via pgxpool,
mapping rows to models.

Repository layer must not contain higher-level business orchestration.

## Current domain model

Core tables:

users
storage_cells
pallets
boxes
products
batches
markers
scan_events
operation_history

Enum:

object_type:
storage_cell
pallet
box
product
batch
## Entity meaning
products

Reference data / nomenclature:

sku
name
unit

Represents: what the product is in general.

batches

A concrete product batch / inventory unit.
Can be linked to:

box_id
pallet_id
storage_cell_id

This is intentionally more flexible than the strict minimum for MVP.

storage_cells

Warehouse cells like:

A-01-01
B-02-03
pallets

A pallet belongs to a storage cell.

boxes

A box may be:

on a pallet, or
directly in a storage cell.
markers

Marker table with:

marker_code
object_type
object_id

This is a polymorphic reference.
Do not try to replace it with regular FKs to every table unless explicitly required.

scan_events

Stores scan facts:

who scanned,
which marker,
device id,
success flag,
timestamp.
operation_history

Stores business operations:

target object,
operation type,
user,
details JSONB,
timestamp.
## Current implemented backend scope

The MVP backend already includes:

object resolution by marker,
unified scan flow,
scan history,
operation history,
move box use case,
Swagger docs,
Docker-based runtime,
working migrations,
demo seed data.

Treat this as an existing system, not a greenfield backend.

---

## Planned next scope: admin QR labels and printing

The next planned backend extension is an admin-oriented label printing flow.

Goal:
- allow an admin to open a simple web page,
- preview labels for warehouse objects,
- generate QR codes from existing `marker_code` values,
- print labels and physically place them on warehouse objects.

Important design rule:
- `marker_code` remains the source of truth;
- QR is only a visual/transport representation of `marker_code`;
- do not store QR image files in the database unless explicitly required.

Recommended implementation direction:
- add backend support for admin label data retrieval;
- support box labels first as the highest-priority scenario;
- allow later extension to storage cells, pallets, batches, and products;
- provide a simple built-in web page or admin print page for preview and printing;
- keep the page minimal and demo-friendly rather than building a large admin system.

Preferred behavior:
- admin can open the print page;
- admin can request labels for boxes;
- backend returns objects with business code and `marker_code`;
- QR is generated from `marker_code`;
- labels are printable in a compact format suitable for stickers.

What to avoid in this scope:
- do not redesign the existing marker model;
- do not replace `marker_code` with binary assets;
- do not make mobile printing the primary flow;
- do not overengineer a full admin platform if a simple printable web page is enough for MVP.

## Existing endpoints
Health
GET /healthz
Objects
GET /api/v1/objects/by-marker?marker_code=...
Scan events
POST /api/v1/scan-events
GET /api/v1/scan-events?limit=...
Operation history
POST /api/v1/operations
GET /api/v1/operations?limit=...
Unified scan
POST /api/v1/scan
Move box
POST /api/v1/boxes/move

When working on the mobile client, these endpoints are the integration contract.

## Current service behavior
object_service
finds a marker by marker_code;
resolves object based on object_type;
loads the proper entity;
builds ObjectCard.
scan_service
accepts marker_code;
resolves the object;
writes a successful scan_event;
returns object + scan_event;
if marker/object is not found, writes failed scan_event with success=false.
move_box_service
accepts:
box_marker_code
to_storage_cell_marker_code
validates that the first marker points to a box;
validates that the second marker points to a storage cell;
loads the box and target storage cell;
updates box placement;
writes operation history;
returns move result.

Important:

moving a box currently means moving it to a storage cell;
repository logic nulls pallet_id and sets storage_cell_id.
## Current implementation notes

Be aware of these existing realities:

Swagger already exists and should be kept aligned with handlers.
Backend wiring already exists in main.go.
Demo data exists via SQL seed.
For a realistic move-box demo, a second storage cell may be needed.
The current move_box flow is acceptable for MVP even if not fully transactional.
For quick demo data, direct SQL/mock data is acceptable; do not force a heavy migration workflow for every tiny demo tweak.
How to work in this repo

When asked to implement or modify backend code, follow this order:

Read the relevant handler.
Read the corresponding service.
Read the related repositories.
Preserve the current interfaces and folder layout.
Make the smallest coherent change.
Keep API contracts stable unless the user explicitly wants contract changes.
Update Swagger comments when endpoint behavior changes.
Prefer consistency with existing naming and style over introducing a new pattern.
What to avoid

Do not:

redesign the entire project;
introduce a new architectural doctrine;
split the monolith;
move business logic into handlers;
move orchestration into repositories;
create giant “universal” repositories;
replace marker polymorphism with a totally different mapping system;
suggest local-only workflows that ignore Docker;
break the mobile client contract casually.
Validation mindset

Before considering backend work done, verify mentally that:

the change fits the current layer boundaries;
handler/service/repository interfaces still make sense;
the mobile client contract is preserved;
Swagger is still truthful;
no unnecessary structural churn was introduced.
Relationship to the mobile client

This repository is the source of truth for the client repository.

If working across both repositories:

read this backend AGENTS.md first;
inspect actual endpoint contracts and models;
then go to the client repository;
implement the client against the existing backend rather than inventing a new API.

The client must adapt to this backend, not the other way around, unless the user explicitly requests backend changes.

---

## Current unfinished work: racks instead of pallets

The user decided to move the active warehouse model away from pallets.
Target active model:

`rack -> storage_cell -> box -> batch -> product`

Business rule:
- a rack groups storage cells and helps find the warehouse sector;
- every storage cell must belong to a rack;
- a storage cell may contain multiple boxes/batches, but only for one product;
- boxes are the main physical unit for received goods;
- batches store the product quantity and must be placed in boxes, not directly in storage cells;
- pallets are legacy and should not be part of the active mobile flow.

Already started:
- added backend rack entity/API;
- added `rack_id` to storage cells;
- added rack QR marker support;
- object cards can resolve rack markers;
- added backend checks so one storage cell cannot contain different products;
- demo seed/migration now moves demo storage to rack `RACK-A-001` and marker `MRK-RACK-001`;
- active content summaries no longer mention pallets;
- label/QR generation supports active object types: rack, storage_cell, box, batch, product.

Still needs to be finished:
- keep `pallets`/`pallet_id` as legacy hidden fields unless the user explicitly decides to drop them;
- verify operation details/history wording after deploy so it does not mention pallets in active flows;
- verify object cards for rack, storage_cell, box, batch, product after the model change;
- run `go test ./...` before considering backend changes ready.
