# 🛡️ TerraLess Host Agent (`terraless-agent`)

[![Go Version](https://img.shields.io/github/go-mod/go-version/AetherRuin/terraless-agent)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build & Release](https://github.com/AetherRuin/terraless-agent/actions/workflows/release.yml/badge.svg)](https://github.com/AetherRuin/terraless-agent/actions)

An open-source, lightweight, high-performance host agent written in **Go** that orchestrates game server instances (Minecraft, Palworld, SteamCMD games) and securely connects host nodes to the **TerraLess Control Plane**.

---

## 🌟 Architectural Features

* **⚡ Ultra-Low Resource Overhead**: Compiled native Go daemon requiring < 15MB RAM footprint.
* **🔒 Zero-Trust Security**:
  * **JWT Authentication**: Performs 1-time token registration, storing non-replayable JWT credentials.
  * **Unprivileged Process Isolation**: Host agent executes as a daemon while running game server processes under an unprivileged user (`minecraft`/`palworld`).
  * **Command Sanitization**: Prevents execution of destructive console commands (`stop`, `kill`, `shutdown`).
* **📡 Real-Time WebSocket Control**: Low-latency bidirectional control loop streaming live console logs, hardware metrics (CPU/RAM), and readiness callbacks to the control plane.
* **🎮 Multi-Game Engine Support**:
  * **Java Engine**: Dynamic JDK path resolution and automatic JVM heap sizing (`-Xms`/`-Xmx`).
  * **SteamCMD Engine**: Native Linux binary process runner for dedicated servers (Palworld, Valheim, Rust, ARK).

---

## 🏗 System Architecture

```
  ┌──────────────────────────────────────────────────────────┐
  │                 TerraLess Control Plane                  │
  │        (Cloudflare Workers & Durable Objects)            │
  └────────────────────────────▲─────────────────────────────┘
                               │
               Secure WebSockets (WSS + JWT Auth)
                               │
  ┌────────────────────────────▼─────────────────────────────┐
  │                   TerraLess Host Agent                   │
  │                  (/usr/local/bin/agent)                  │
  ├──────────────────────────────────────────────────────────┤
  │  • Process Supervision    • Log Streamer & Sanitizer     │
  │  • Hardware Stats Loop    • Systemd Integration          │
  └────────────────────────────┬─────────────────────────────┘
                               │
               Supervises Subprocesses as 'minecraft' user
                               │
          ┌────────────────────┴────────────────────┐
          │                                         │
┌─────────▼──────────┐                   ┌──────────▼─────────┐
│ Minecraft (Java)   │                   │ Palworld (SteamCMD)│
│ server.jar         │                   │ PalServer.sh       │
└────────────────────┘                   └────────────────────┘
```

---

## 🚀 Quickstart Installation

Run the one-line automated installer on any systemd-enabled Linux host (`amd64` or `arm64`):

```bash
curl -sSL https://raw.githubusercontent.com/AetherRuin/terraless-agent/main/install.sh | bash
```

---

## ⚙️ Configuration Reference

The agent configuration file is located at `/etc/aetherruin-agent/config.json`:

```json
{
  "serverId": "node-us-east-01",
  "registrationToken": "reg_secret_token_12345",
  "workerUrl": "wss://control.terraless.io/ws",
  "gameId": "minecraft",
  "javaPath": "/usr/bin/java",
  "serverJar": "/opt/game/server.jar",
  "maxMemory": "2048M",
  "minMemory": "1024M"
}
```

### Configuration Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `serverId` | `string` | **Required**. Unique host node identifier. |
| `registrationToken` | `string` | **Required**. Initial authorization token for control plane onboarding. |
| `workerUrl` | `string` | **Required**. Control plane WebSocket URL. |
| `gameId` | `string` | **Required**. Game engine type (`"minecraft"` or `"palworld"`). |
| `agentJwt` | `string` | *Auto-populated*. Persisted JWT issued after successful 1-time registration. |
| `javaPath` | `string` | *Optional*. Explicit JDK path. Defaults to system `$PATH` resolution. |
| `maxMemory` | `string` | *Optional*. Java heap maximum allocation (e.g. `"2048M"`). Defaults to `"1024M"`. |
| `minMemory` | `string` | *Optional*. Java heap initial allocation (e.g. `"1024M"`). Defaults to `"512M"`. |

---

## 🔨 Building & Development

### Requirements
* **Go** 1.22+

### Local Build & Test
```bash
# Clone the repository
git clone https://github.com/AetherRuin/terraless-agent.git
cd terraless-agent

# Run unit tests
go test -v ./...

# Compile agent binary
go build -o dist/terraless-agent .
```

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for details.
