# Raspberry Pi Setup

Step-by-step guide for setting up a Raspberry Pi to run the mpr-6zhmaut API.

## Hardware

- **Raspberry Pi 3 Model B** (or newer) -- any Pi with USB ports works
- **USB-to-serial adapter** (FTDI, CH340, or PL2303) -- connects to the amp's RS-232 port
- **Monoprice MPR-6ZHMAUT amplifier(s)** -- one or more, daisy-chained via the serial bus
- **RS-232 cable** -- from USB-serial adapter to the amp's DB-9 serial port
- **Power supply** for the Pi
- **MicroSD card** (8GB+ recommended)
- **Ethernet cable or WiFi** for network access

### Wiring

```
Pi USB port ──→ USB-Serial Adapter ──→ RS-232 Cable ──→ Amp 1 Serial Port
                                                              │
                                                        Amp 2 Serial Port
                                                        (daisy-chained)
```

## Flash the OS

1. Download and install [Raspberry Pi Imager](https://www.raspberrypi.com/software/)
2. Insert the MicroSD card into your computer
3. Open Raspberry Pi Imager
4. Select device: **Raspberry Pi 3**
5. Select OS: **Raspberry Pi OS (Legacy, 32-bit) Lite** (Debian Bookworm, no desktop)
6. Select your MicroSD card as the storage target

### Customisation (before writing)

In the Raspberry Pi Imager customisation step:

- **Set hostname**: `mpr` (so the Pi is reachable at `mpr.local` on your network)
- **Enable SSH**: check "Use password authentication" or add your public key
- **Set username and password**: choose your credentials (e.g., `cbschuld`)
- **Configure WiFi**: enter your network SSID and password (or use ethernet)
- **Set locale**: your timezone and keyboard layout

7. Click **Write** and wait for the flash to complete
8. Insert the MicroSD card into the Pi and power it on

## First Boot

Wait about 60 seconds for the Pi to boot, then connect via SSH:

```bash
ssh <username>@mpr
```

If `mpr` doesn't resolve, find the Pi's IP address from your router's admin page and use that instead.

### Set up SSH key authentication (recommended)

From your Mac/PC:

```bash
# Copy your SSH key to the Pi (prompts for password once)
ssh-copy-id -i ~/.ssh/id_ed25519.pub <username>@mpr

# Verify passwordless login
ssh <username>@mpr
```

### Verify the serial adapter

Plug in the USB-to-serial adapter and check that it's recognized:

```bash
ls -la /dev/ttyUSB*
# Should show: /dev/ttyUSB0

# Check kernel messages for the adapter
dmesg | grep -i usb | tail -10
```

### Verify dialout group membership

Your user must be in the `dialout` group to access the serial port:

```bash
groups
# Should include 'dialout' in the list

# If not, add it:
sudo usermod -aG dialout $USER
# Log out and back in for it to take effect
```

## Deploy the API

From your development machine:

```bash
cd ~/Projects/mpr-6zhmaut-golang-api

# Build for the Pi (cross-compile)
make build-pi

# Deploy (copies binary + service file, restarts the service)
PI_HOST=<username>@mpr make deploy
```

Or manually:

```bash
# Copy the binary
scp mpr-api-linux-armv7 <username>@mpr:/tmp/mpr-api

# Copy the systemd service file
scp mpr-api.service <username>@mpr:/tmp/mpr-api.service

# SSH in and install
ssh <username>@mpr
sudo mv /tmp/mpr-api /usr/local/bin/mpr-api
sudo chmod +x /usr/local/bin/mpr-api
sudo mv /tmp/mpr-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable mpr-api
sudo systemctl start mpr-api
```

## Verify it's running

```bash
# Check service status
sudo systemctl status mpr-api

# Watch the logs
journalctl -u mpr-api -f
```

You should see:
1. `starting mpr-6zhmaut-api` with your configuration
2. `state transition DISCONNECTED -> PROBING`
3. `probing baud rate 115200`
4. `probe success` at the current baud rate
5. `state transition PROBING -> READY`
6. `http server listening port 8181`

### Test from your network

From any device on the same network:

```bash
# Get all zone statuses
curl http://mpr:8181/api/zones | jq

# Check health
curl http://mpr:8181/api/health | jq

# Open the web UI in a browser
open http://mpr:8181/
```

## Configure the systemd service

The service file is at `/etc/systemd/system/mpr-api.service`. Edit the `Environment=` lines to match your setup:

```ini
Environment=DEVICE=/dev/ttyUSB0
Environment=TARGET_BAUDRATE=115200
Environment=PORT=8181
Environment=AMPCOUNT=2              # 1, 2, or 3
Environment=POLL_INTERVAL=5s
Environment=HEALTH_INTERVAL=30s
Environment=LOG_LEVEL=info          # debug, info, warn, error
```

After editing:

```bash
sudo systemctl daemon-reload
sudo systemctl restart mpr-api
```

## mDNS / Avahi service registration (for iOS app)

The Pi already advertises itself as `mpr.local` via Avahi. To additionally advertise the API service for automatic discovery by the iOS app:

```bash
sudo tee /etc/avahi/services/mpr-api.service << 'EOF'
<?xml version="1.0" standalone='no'?>
<!DOCTYPE service-group SYSTEM "avahi-service.dtd">
<service-group>
  <name>MPR Audio Controller</name>
  <service>
    <type>_mpr-audio._tcp</type>
    <port>8181</port>
  </service>
</service-group>
EOF

sudo systemctl restart avahi-daemon
```

## Troubleshooting

### Service won't start

```bash
# Check for errors
journalctl -u mpr-api --no-pager -n 50

# Common issues:
# - /dev/ttyUSB0 not found → USB-serial adapter not plugged in
# - Permission denied on /dev/ttyUSB0 → user not in dialout group
# - Port 8181 already in use → another service on that port
```

### Amp not responding

```bash
# Check if serial adapter is recognized
ls -la /dev/ttyUSB*

# Check if amp is powered on and serial cable is connected
# The service will probe all baud rates and retry automatically

# Watch the probing sequence
journalctl -u mpr-api -f
```

### After amp power cycle

The service handles this automatically. When the amp resets to 9600 baud:
1. The health monitor detects the timeout
2. Recovery begins: probes baud rates starting with the target
3. Falls back to 9600, steps back up to the target
4. Returns to READY state

Watch the recovery in real-time:
```bash
journalctl -u mpr-api -f
# or
curl http://mpr:8181/api/health/events | jq
```
