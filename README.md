# lincmox

LincStation LED control for Proxmox VE — Go rewrite.

```
██╗     ██╗███╗   ██╗ ██████╗███╗   ███╗ ██████╗ ██╗  ██╗
██║     ██║████╗  ██║██╔════╝████╗ ████║██╔═══██╗╚██╗██╔╝
██║     ██║██╔██╗ ██║██║     ██╔████╔██║██║   ██║ ╚███╔╝
██║     ██║██║╚██╗██║██║     ██║╚██╔╝██║██║   ██║ ██╔██╗
███████╗██║██║ ╚████║╚██████╗██║ ╚═╝ ██║╚██████╔╝██╔╝ ██╗
╚══════╝╚═╝╚═╝  ╚═══╝ ╚═════╝╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═╝
```

Controls the LEDs of a **LincStation N1** from a Proxmox VE host via I2C.

Two binaries are provided:

| Binary | Package | Description |
|---|---|---|
| `lincmox` | `lincmox` | CLI — direct I2C access or proxy to daemon |
| `lincmoxd` | `lincmoxd` | Daemon — REST API + Web UI + background monitors |

---

## Installation

### From the LincMox APT repository (recommended)

```bash
curl -fsSL https://repo.lincmox.ovh/public.key | sudo gpg --dearmor -o /usr/share/keyrings/lincmox.gpg
echo "deb [signed-by=/usr/share/keyrings/lincmox.gpg] https://repo.lincmox.ovh trixie main" \
    | sudo tee /etc/apt/sources.list.d/lincmox.list
sudo apt update

# CLI only
sudo apt install lincmox

# Daemon (includes CLI)
sudo apt install lincmoxd
```

### From source

Requires **Go 1.22+**.

```bash
git clone https://gitea.stela.ovh/Lincmox/lincmox.git
cd lincmox
go build -o bin/lincmox   ./cmd/lincmox
go build -o bin/lincmoxd  ./cmd/lincmoxd
```

---

## CLI Usage — `lincmox`

When `lincmoxd` is running, `lincmox` automatically routes all commands through the daemon's
Unix socket (`/run/lincmoxd.sock`). No configuration needed.

### LED control

```bash
# Power LED
lincmox led power on white
lincmox led power off red
lincmox led power blink on
lincmox led power blink off

# SATA LEDs (1-2)
lincmox led sata 1 on white
lincmox led sata 2 off red
lincmox led sata 1 blink on

# NVMe LEDs (1-4)
lincmox led nvme 1 on orange
lincmox led nvme 4 off white
lincmox led nvme 2 blink on

# Network LED
lincmox led network on white
lincmox led network off red
lincmox led network blink on
```

Available colors: `white`, `red`, `orange`

### LED Strip

```bash
# Animation mode
lincmox strip animation off
lincmox strip animation breath
lincmox strip animation loop

# Brightness (0-255)
lincmox strip brightness 128

# Color (R G B, 0-255 each)
lincmox strip color 255 0 0

# Loop colors (for "loop" animation)
lincmox strip loop 1 color 0 255 0
lincmox strip loop 2 color 0 0 255
```

### Reset

```bash
lincmox reset full    # reset all LEDs and strip
lincmox reset leds    # reset LEDs only
lincmox reset strip   # reset strip only
```

### Status

```bash
lincmox status
```

### Simulation mode (no hardware)

```bash
lincmox --simulate led power on white
# or
LINCMOX_ENV=dev lincmox led power on white
```

### Debug / Verbose mode

Use the `--verbose` flag to debug I2C communication. This will log the auto-detected bus, the target register, the hex values, and the result of the I2C transfer.

```bash
lincmox --verbose led network on white
```

### Forcing the I2C bus

By default `lincmox` auto-detects the bus by probing `/dev/i2c-0` through `/dev/i2c-9`
for a device at address `0x26`. If auto-detection fails but you know the device is
present (e.g. it shows up in `i2cdetect -y <bus>`), you can force the bus number:

```bash
lincmox --bus 4 led network on white
lincmoxd --bus 4
```

Combine with `--verbose` to confirm which bus was used.

---

## Daemon Usage — `lincmoxd`

### Start

```bash
# With systemd (after apt install lincmoxd)
sudo systemctl enable --now lincmoxd

# Manual
lincmoxd --addr :8080 --sock /run/lincmoxd.sock

# Simulation mode
lincmoxd --simulate
```

### REST API

Base URL: `http://<host>:8080` or via Unix socket at `/run/lincmoxd.sock`.

#### LEDs

```
POST /api/v1/led/power/on           {"color": "white"}
POST /api/v1/led/power/off          {"color": "red"}
POST /api/v1/led/power/blink        {"blink": true}
POST /api/v1/led/sata/1/on          {"color": "white"}
POST /api/v1/led/sata/2/off         {"color": "red"}
POST /api/v1/led/nvme/1/on          {"color": "orange"}
POST /api/v1/led/nvme/4/blink       {"blink": false}
POST /api/v1/led/network/on         {"color": "white"}
```

#### Strip

```
POST /api/v1/strip/animation        {"mode": "breath"}
POST /api/v1/strip/brightness       {"value": 128}
POST /api/v1/strip/color            {"r": 255, "g": 0, "b": 0}
POST /api/v1/strip/loop/1/color     {"r": 0, "g": 255, "b": 0}
POST /api/v1/strip/loop/2/color     {"r": 0, "g": 0, "b": 255}
```

#### Reset & Status

```
POST /api/v1/reset                  {"mode": "full"}
GET  /api/v1/status
```

#### Monitors

```
GET  /api/v1/monitors
POST /api/v1/monitors/network/enable   {"iface": "eth0", "interval_ms": 500, "threshold_bytes": 1024}
POST /api/v1/monitors/network/disable
```

### Web UI

Open `http://<host>:8080` in your browser.

---

## Monitors

Monitors are background goroutines in `lincmoxd` that drive LEDs automatically based on
system metrics.

| Monitor | Source | LED | Description |
|---|---|---|---|
| `network` | `/sys/class/net/<iface>/statistics/` | NetworkLED | Blinks when traffic exceeds threshold |
| `disk` | `/sys/block/<dev>/stat` | SATA/NVMe LEDs | *(coming soon)* |

**Priority**: manual commands from the CLI or API take precedence for 30 seconds, then
monitors resume automatically.

---

## Cohabitation: CLI + Daemon

| Scenario | Behavior |
|---|---|
| `lincmox` only | Direct I2C access |
| `lincmoxd` only | REST API + Web UI, no CLI |
| Both installed, daemon **running** | CLI routes transparently through Unix socket — no I2C conflict |
| Both installed, daemon **stopped** | CLI falls back to direct I2C access |

---

## Architecture

```
lincmox/
  internal/lincstation/   shared driver (I2C + LED control logic)
  cmd/lincmox/            CLI binary
  cmd/lincmoxd/           daemon binary (REST API + Web UI + monitors)
  web/dist/               embedded Web UI assets
```

The `internal/` package boundary ensures the driver cannot be imported by external Go modules.

---

## Hardware

- **Device**: LincStation N1
- **Interface**: I2C
- **I2C address**: `0x26`
- **Host OS**: Proxmox VE (Debian Trixie, amd64)

---

## Development

```bash
# Run tests
go test ./...

# Build both binaries
go build ./cmd/lincmox ./cmd/lincmoxd

# Simulate without hardware
LINCMOX_ENV=dev ./bin/lincmox led power on white
./bin/lincmoxd --simulate
```

---

## Versioning

This project uses [Semantic Versioning](https://semver.org/) with Conventional Commits.

**Commit prefixes:**
- `[FEAT]-` — new feature
- `[FIX]-` — bug fix
- `[CHORE]-` — build, CI, dependencies
- `[DOC]-` — documentation

**Release tags:**
```bash
git tag v2.0.0      # stable release
git tag v2.0.0-rc1  # release candidate (→ testing channel)
```

---

## Deploying

The project is built and packaged automatically by **Gitea Actions** (Gitea CI/CD).
Pushing to the `dev` branch or tagging a release triggers a pipeline that produces
Debian packages and publishes them to the LincMox repositories.

### 1. Pushing to `dev` (snapshot)

Any push to the `dev` branch builds a **snapshot** package:

```
2.0.0~dev+<short-sha>
```

- **Channel**: `testing`
- It is uploaded to the **Gitea Debian registry** (`trixie/testing`).
- Old `~dev+` packages are automatically cleaned up (only the current one is kept).

### 2. Tagging a release (stable or release candidate)

```bash
git tag v2.0.0      # stable release
git push origin v2.0.0
```

The workflow `determine-build-env` extracts the version, channel and Debian branch:

| Ref | Package version | Channel | Debian branch |
|---|---|---|---|
| push on `dev` | `2.0.0~dev+<sha>` | `testing` | `dev` |
| tag `v2.0.0-rc1` | `2.0.0~rc1` | `testing` | `dev` |
| tag `v2.0.0` | `2.0.0` | `main` | `dev` |

Any tag containing `alpha`, `beta`, `rc` or `dev` → **testing**; otherwise → **main**.

### Workflow output (`.gitea/workflows/build.yml`)

1. `gbp dch` generates the Debian changelog from commits.
2. `dpkg-buildpackage` builds two binary packages:
   - **`lincmox_<version>_amd64.deb`** — CLI (`/usr/bin/lincmox`)
   - **`lincmoxd_<version>_amd64.deb`** — daemon + REST API + Web UI
     (`/usr/bin/lincmoxd`, systemd unit)
3. `push-to-registry` uploads all `.deb` files to the **Gitea Debian registry**
   (pool `trixie/<channel>`).
4. On **main** channel only, `publish-to-prod-repository` commits the `.deb` files
   into **`lincmox_repo`** (`packages/main/`), which rebuilds and redeploys the
   production APT server (`repo.lincmox.ovh`).

Install from the production APT server:
```bash
sudo apt install lincmox lincmoxd
```

---

## Acknowledgements

## Acknowledgements

Thanks to the reverse engineering work from:
- <https://github.com/tsew/lincstation_leds>
- <https://gist.github.com/aluevano/ca6431f4f15d8ea62df57e67df7d4c3d>
- <https://github.com/fazalmajid/lincstation_leds>

