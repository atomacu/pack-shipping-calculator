# Pack Shipping Calculator

Application for calculating the minimum complete packs required to fulfil an item order.

The product has two deployable parts:

- `backend/` - Go HTTP API, SQLite persistence, and packing domain logic.
- `frontend/` - React UI that calls the backend API.

## Live demo URLs:

- UI: https://atomacu.github.io/pack-shipping-calculator/
- API: https://pack-shipping-calculator.onrender.com

## Quick Start

Run the full app with Docker:

```sh
docker compose up --build
```

Open the UI:

```text
http://localhost:3000
```

The backend API is also exposed locally:

```text
http://localhost:8080
```

Reset local Docker data, including persisted pack sizes:

```sh
docker compose down -v
```

## Local Development

Make is optional. If it is installed, these shortcuts wrap the same Docker workflow:

```sh
make dev
make validate
```

Without Make, use Docker directly:

```sh
docker compose up --build
docker compose down
docker compose down -v
```

Useful Make targets:

```sh
make dev       # clean Docker state, build, and run the app
make validate  # clean Docker state, run tests/coverage/vet, build images, validate Compose
make down      # stop the app
make reset     # stop the app and remove persisted Docker data
make clean     # remove generated local artifacts and local Docker images
```

## Pack Sizes

Default seed pack sizes:

- 250
- 500
- 1000
- 2000
- 5000

Pack sizes are configurable from the UI and persisted by the backend in SQLite. The config file only provides initial seed values when the database is empty. Runtime values stored in SQLite are authoritative after startup.

Orders must be whole numbers from `1` to `1000000`. The limit keeps the dynamic-programming calculator bounded and returns a structured JSON validation error instead of exhausting server memory.

## GitHub Actions

The repository includes three workflows:

- `CI` runs `make validate` on pull requests and pushes to `main`. This runs backend tests with 100% coverage, `go vet`, Docker image builds, and Compose validation.
- `Deploy Frontend` builds the React app in Docker and deploys `frontend/dist` to GitHub Pages.
- `Deploy Backend` optionally triggers a Render backend deploy hook.


## API Examples

Health:

```sh
curl http://localhost:8080/healthz
```

Get configured pack sizes:

```sh
curl http://localhost:8080/api/v1/packs
```

Replace pack sizes:

```sh
curl -X PUT http://localhost:8080/api/v1/packs \
  -H 'Content-Type: application/json' \
  -d '{"pack_sizes":[500,250,500,1000]}'
```

Calculate an order:

```sh
curl http://localhost:8080/api/v1/orders/calculate \
  -H 'Content-Type: application/json' \
  -d '{"items":12001}'
```

Example response:

```json
{
  "items_ordered": 12001,
  "items_shipped": 12250,
  "items_over": 249,
  "total_packs": 4,
  "packs": [
    { "size": 5000, "quantity": 2 },
    { "size": 2000, "quantity": 1 },
    { "size": 250, "quantity": 1 }
  ]
}
```

## Architecture

Backend layers:

- `backend/cmd/api` - process entry point.
- `backend/internal/app` - startup wiring.
- `backend/internal/config` - JSON config loading.
- `backend/internal/httpapi` - HTTP routes, DTOs, JSON errors.
- `backend/internal/packs` - pack-size validation, persistence orchestration, and calculation orchestration.
- `backend/internal/packing` - pure packing algorithm.
- `backend/internal/storage/sqlite` - SQLite repository.

The frontend is only an API client. Packing logic is not duplicated in the UI.
