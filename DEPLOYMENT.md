# Deployment

This repository is deployed to a VPS via GitHub Actions.

## Workflow

The workflow file is:

- `.github/workflows/backend-ci-cd.yml`

On every push to `main`, GitHub Actions:

1. runs `go test ./...`;
2. builds the API binary from `./cmd/api`;
3. connects to the VPS over SSH;
4. uploads the project files to the server;
5. runs `docker-compose up -d --build --remove-orphans`;
6. prunes unused Docker containers, images, and build cache.

## Required GitHub Secrets

Configure these repository secrets in GitHub:

- `VPS_HOST` - server IP or hostname
- `VPS_PORT` - SSH port, usually `22`
- `VPS_USER` - SSH user, for example `root`
- `VPS_SSH_PRIVATE_KEY` - private SSH key used by GitHub Actions
- `VPS_APP_DIR` - absolute path to the backend repository on the server

For the current VPS setup the values are expected to be similar to:

- `VPS_HOST=159.194.222.17`
- `VPS_PORT=22`
- `VPS_USER=root`
- `VPS_APP_DIR=/root/Warehouse_service`

## First-time server preparation

The server must already have:

- Docker installed
- Docker Compose available as `docker-compose` or `docker compose`
- a writable target directory for deployment

## Notes

- The workflow deploys branch `main`.
- The deploy job uploads repository contents from GitHub Actions to the server over SSH.
- `.git`, `.github`, `.gocache`, and `.idea` are excluded from upload.
- If `.env` is tracked in git, the uploaded version will replace the server copy during deployment.
- PostgreSQL is intentionally not published to the public internet by default.
- The deploy job prunes:
  - unused containers
  - unused images
  - build cache
- Before restart, the deploy job force-removes old `api` and `migrate` containers to avoid legacy `docker-compose` recreate issues on the VPS.
- Docker volumes are not pruned automatically, so PostgreSQL data is preserved.
