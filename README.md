# mpr-6zhmaut-golang-api

A self-healing REST API for the [Monoprice MPR-6ZHMAUT](https://www.monoprice.com/product?p_id=10761) 6-zone home audio amplifier, written in Go. Supports multiple daisy-chained amplifiers (up to 18 zones) and automatically recovers from amp power cycles by detecting connection loss, dropping to the default baud rate, and stepping back up to the target speed.

Successor to [jnewland/mpr-6zhmaut-api](https://github.com/jnewland/mpr-6zhmaut-api) (Node.js).

## Features

- **Auto-recovery** -- detects when amps reset (power cycle, breaker trip), reconnects at 9600 baud, and steps back up to the target baud rate (e.g., 115200)
- **Optimistic state with background polling** -- GET requests serve instantly from an in-memory cache; a background poller catches physical keypad changes within seconds
- **`?live=true` override** -- bypass the cache on any GET to query the amp directly
- **Serialized command queue** -- all serial port access goes through a FIFO queue with timeouts, eliminating race conditions
- **Connection state machine** -- explicit lifecycle states (`DISCONNECTED` → `PROBING` → `NEGOTIATING` → `READY` → `RECOVERING`) prevent impossible operations
- **Rich health endpoints** -- `/health` exposes connection state, baud rate, cache age, queue stats, recovery history; `/health/events` shows a ring buffer of the last 100 system events
- **Structured JSON logging** -- every component logs via `slog` for easy debugging with `journalctl`
- **Multi-amp support** -- 1-3 daisy-chained amplifiers (6-18 zones)
- **Embedded web UI** -- React 19 + Vite SPA built into the binary via `embed.FS`, served from the same port as the API
- **Single binary** -- compiles to one static binary with the web UI baked in, runs as a systemd service

## Quick Start

```bash
# Build everything (web UI + Go binary)
make build

# Or build for Raspberry Pi 3B
make build-pi

# Deploy to Pi (set PI_HOST)
PI_HOST=mpr make deploy

# Run locally (defaults: /dev/ttyUSB0, 115200 baud, port 8181, 1 amp)
AMPCOUNT=2 ./mpr-api
```

### Manual Build

```bash
# Build web UI
cd web && npm install && npm run build && cd ..

# Copy build output for embedding
cp -r web/dist cmd/mpr-api/dist

# Build Go binary
go build -o mpr-api ./cmd/mpr-api/

# Cross-compile for Raspberry Pi
GOOS=linux GOARCH=arm GOARM=7 go build -o mpr-api ./cmd/mpr-api/    # Pi 3/4 (32-bit)
GOOS=linux GOARCH=arm64 go build -o mpr-api ./cmd/mpr-api/           # Pi 4/5 (64-bit)
```

## API

All endpoints are available at both `/api/...` (for the web/iOS clients) and `/...` (legacy, backward compatible). All GET endpoints accept `?live=true` to bypass the cache and query the amp directly.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/zones` | All zones (from cache) |
| `GET` | `/api/zones/:zone` | Single zone status |
| `GET` | `/api/zones/:zone/:attribute` | Single attribute value (plain text) |
| `POST` | `/api/zones/:zone/:attribute` | Set attribute (body = value), returns updated zone |
| `GET` | `/api/health` | Connection state, baud rate, cache age, queue stats |
| `GET` | `/api/health/events` | Last 100 system events (state changes, recoveries, errors) |

The web UI is served at `/` from the same port.

### Zone IDs

Zones are numbered by amp and position: amp 1 = `11`-`16`, amp 2 = `21`-`26`, amp 3 = `31`-`36`.

### Attributes

| Name | Code | Description |
|------|------|-------------|
| `power` | `pr` | Power on/off (`00`/`01`) |
| `mute` | `mu` | Mute (`00`/`01`) |
| `volume` | `vo` | Volume level |
| `treble` | `tr` | Treble |
| `bass` | `bs` | Bass |
| `balance` | `bl` | Balance |
| `channel` / `source` | `ch` | Input source |
| `keypad` | `ls` | Keypad status |

### Examples

```bash
# Get all zones
curl http://localhost:8181/zones

# Get zone 11 volume (from cache)
curl http://localhost:8181/zones/11/volume

# Get zone 11 volume (live from amp)
curl http://localhost:8181/zones/11/volume?live=true

# Set zone 11 volume to 15
curl -X POST -d '15' http://localhost:8181/zones/11/volume

# Turn on zone 21
curl -X POST -d '01' http://localhost:8181/zones/21/power

# Check system health
curl http://localhost:8181/health

# View recent events (state changes, recoveries)
curl http://localhost:8181/health/events
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DEVICE` | `/dev/ttyUSB0` | Serial port device path |
| `TARGET_BAUDRATE` | `115200` | Target operating baud rate |
| `PORT` | `8181` | HTTP listen port |
| `AMPCOUNT` | `1` | Number of daisy-chained amps (1-3) |
| `CORS` | `false` | Enable CORS headers |
| `POLL_INTERVAL` | `5s` | Background zone polling interval |
| `HEALTH_INTERVAL` | `30s` | Connection health check interval |
| `CMD_TIMEOUT` | `2s` | Serial command timeout |
| `STEP_DELAY` | `500ms` | Delay between baud rate step-up commands |
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

## Deployment (systemd)

```bash
# Build for target platform
GOOS=linux GOARCH=arm64 go build -o mpr-api ./cmd/mpr-api/

# Copy binary and service file to the Pi
scp mpr-api pi@<pi-ip>:/usr/local/bin/
scp mpr-api.service pi@<pi-ip>:/etc/systemd/system/

# On the Pi
sudo systemctl daemon-reload
sudo systemctl enable mpr-api
sudo systemctl start mpr-api

# View logs
journalctl -u mpr-api -f
```

The included `mpr-api.service` file configures `Restart=always` with systemd hardening (NoNewPrivileges, ProtectSystem, PrivateTmp). Edit the `Environment=` lines to match your setup.

## How Recovery Works

1. The **health monitor** sends a probe command every 30 seconds
2. If the amp doesn't respond (timeout) or returns `Command Error.`, recovery begins
3. The state machine transitions: `READY` → `RECOVERING` → `PROBING`
4. The service tries the **target baud rate first** (maybe the app restarted, not the amp)
5. If that fails, it drops to **9600** (the amp's power-on default) and probes
6. Once found, it **steps up** through intermediate baud rates (9600 → 19200 → 38400 → 57600 → 115200), verifying communication at each step
7. Back to `READY` -- the background poller resumes, API requests are served again

During recovery, all API endpoints return `503 Service Unavailable`. The `/health` endpoint always responds, showing the current state.

## Architecture

```
HTTP clients ──→ REST API ──→ Zone Cache (serves GET instantly)
                    │              ↑
                    ▼              │ updates
                Controller ───────┘
                    │
        ┌───────┬──┴──┬──────────┐
        │       │     │          │
    State    Command  Health   Background
    Machine  Queue    Monitor  Poller
        │       │     │          │
        └───────┴─────┴──────────┘
                    │
                Serial Port
                    │
              Amp 1 ── Amp 2
```

## License

MIT
