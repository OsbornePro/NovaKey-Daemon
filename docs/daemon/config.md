# Configuration

NovaKey-Daemon supports YAML (**preferred**) or JSON configuration files:

- `server_config.yaml`
- `server_config.yml`
- `server_config.json`

The daemon loads configuration from its **WorkingDirectory**.
This directory is set by the installer and differs by platform.

Relative paths such as `devices.json`, `server_keys.json`, and `./logs`
are resolved relative to this directory.

---

## Default WorkingDirectory by platform

### Windows

Installed per-user under LocalAppData.

- **WorkingDirectory:**  
  `%LOCALAPPDATA%\NovaKey\data`
- **Config file:**  
  `%LOCALAPPDATA%\NovaKey\data\server_config.yaml`
- **Runtime data:**  
  `devices.json`, `server_keys.json`, `logs\`

The Scheduled Task created by the installer explicitly sets this directory.

---

### macOS

Installed per-user using a LaunchAgent.

- **WorkingDirectory:**  
  `~/.local/share/novakey`
- **Config file:**  
  `~/.config/novakey/server_config.yaml`
- **Runtime data:**  
  `devices.json`, `server_keys.json`, `logs/`

The installer copies the config into both locations; the daemon runs from
the data directory.

---

### Linux

Installed as a **systemd user service**.

- **WorkingDirectory:**  
  `~/.local/share/novakey`
- **Config file:**  
  `~/.config/novakey/server_config.yaml`
- **Runtime data:**  
  `devices.json`, `server_keys.json`, `logs/`

System packages install a default config under `/usr/share/novakey`,
which is copied into the user’s config directory on first run.

---

## Config file selection order

If multiple config files exist in the WorkingDirectory, NovaKey uses:

1. `server_config.yaml`
2. `server_config.yml`
3. `server_config.json`

