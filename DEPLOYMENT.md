# Deployment

This repository is deployed to a VPS via GitHub Actions.

## Workflow

The workflow file is:

- `.github/workflows/backend-ci-cd.yml`

On every push to `main`, GitHub Actions:

1. runs `go test ./...`;
2. builds the API binary from `./cmd/api`;
3. connects to the VPS over SSH;
4. updates the repository on the server;
5. runs `docker-compose up -d --build`.

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
- this repository cloned once, or permissions to clone it during deploy

## Notes

- The workflow deploys branch `main`.
- The deploy job uses `git pull --ff-only origin main` on the server.
- If server files were changed manually and diverged from `main`, deployment will fail until the server repository is cleaned up.
- If `.env` is tracked in git, updates will also be pulled during deployment.
