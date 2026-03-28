# Development Guide

How to develop, build, and deploy updates to the mpr-6zhmaut API.

## Prerequisites

- **Go 1.21+** -- `brew install go` (macOS) or [go.dev/dl](https://go.dev/dl/)
- **Node.js 18+** -- `brew install node` (macOS) or [nodejs.org](https://nodejs.org/)
- **Make** -- included on macOS and Linux
- **SSH access** to the Raspberry Pi (see [Pi Setup](raspberry-pi-setup.md))

## Project Layout

```
mpr-6zhmaut-golang-api/
  cmd/mpr-api/main.go       # Go entry point, embeds web/dist/
  internal/                  # Go packages (config, serial, amp, api)
  web/                       # Vite + React 19 + TypeScript frontend
    src/                     # React source code
    dist/                    # Vite build output (embedded into Go binary)
  docs/                      # This documentation
  Makefile                   # Build and deploy targets
  mpr-api.service            # systemd unit file for the Pi
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build web UI + Go binary for current platform |
| `make build-web` | Build only the React web UI |
| `make build-go` | Build only the Go binary (requires web UI to be built first) |
| `make build-pi` | Cross-compile for Raspberry Pi 3B (linux/arm/v7) |
| `make deploy` | Build for Pi and deploy via SSH (set `PI_HOST`) |
| `make test` | Run Go tests |
| `make dev` | Run Go binary locally with placeholder web UI |
| `make clean` | Remove all build artifacts |

## Local Development

### Web UI (React)

For frontend development with hot reload:

```bash
cd web
npm install
npm run dev
```

This starts the Vite dev server (usually on port 5173). To proxy API calls to the Pi, add to `web/vite.config.ts`:

```typescript
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://mpr:8181',
    },
  },
});
```

Now `http://localhost:5173/` serves the React app, and all `/api/` requests are forwarded to the Pi.

### Go API

To work on the Go API locally (without a physical amp), you can run with debug logging:

```bash
LOG_LEVEL=debug make dev
```

Note: without a serial device connected, the service will fail at the connection probe step. For Go development without hardware, focus on writing and running unit tests.

### Running tests

```bash
# Go tests
make test
# or
go test ./... -v

# Web linting
cd web && npm run lint
```

## Building

### Full build (web + Go)

```bash
make build
```

This:
1. Runs `npm install && npm run build` in `web/`
2. Copies `web/dist/` to `cmd/mpr-api/dist/`
3. Runs `go build -o mpr-api ./cmd/mpr-api/`

The resulting `mpr-api` binary has the React app embedded.

### Cross-compile for Pi

```bash
make build-pi
```

Produces `mpr-api-linux-armv7` (statically linked, no dependencies on the Pi).

## Deploying Updates

### Using the Makefile

```bash
PI_HOST=<username>@mpr make deploy
```

This builds for the Pi, copies the binary and service file via SCP, and restarts the systemd service -- all in one command.

### Manual deployment

```bash
# Build
make build-pi

# Copy
scp mpr-api-linux-armv7 <username>@mpr:/tmp/mpr-api

# SSH in and swap the binary
ssh <username>@mpr
sudo systemctl stop mpr-api
sudo mv /tmp/mpr-api /usr/local/bin/mpr-api
sudo chmod +x /usr/local/bin/mpr-api
sudo systemctl start mpr-api

# Verify
journalctl -u mpr-api -f
```

### Checking deployment health

After deploying, verify the service is healthy:

```bash
# From your development machine
curl http://mpr:8181/api/health | jq

# Check the state is READY
# Check the cache has 12 zones (for 2 amps)
# Check total_timeouts and total_errors are low
```

## Making Changes

### Adding a new API endpoint

1. Add the handler method to `internal/api/server.go`
2. Register the route in the `routes()` method (add both `/api/...` and `/...` versions)
3. Run `go test ./...` to verify nothing is broken
4. Deploy: `PI_HOST=<username>@mpr make deploy`

### Modifying the web UI

1. Edit files in `web/src/`
2. Preview locally: `cd web && npm run dev` (with Vite proxy to Pi)
3. When ready, deploy: `PI_HOST=<username>@mpr make deploy`

### Modifying serial protocol handling

1. Edit `internal/serial/protocol.go`
2. Update tests in `internal/serial/protocol_test.go`
3. Run `go test ./internal/serial/ -v`
4. Deploy and verify with real hardware

## Git Workflow

```bash
# Make changes
git add -A
git commit -m "description of changes"
git push

# Deploy to Pi
PI_HOST=<username>@mpr make deploy
```

The Pi always runs whatever binary was last deployed -- it doesn't pull from git. Deployment is always an explicit `make deploy` step.
