# Configuration

NovaKey-Daemon supports YAML (*preferred*) or JSON foramtted configuration file  
`server_config.yaml` or `server_config.json`

## Core networking & limits. 
Best practice is to set the private IP address of your computer as the listener instead of `0.0.0.0`.
- `listen_addr` (default `0.0.0.0:60768`)
  - LAN use: bind to `0.0.0.0:60768` or a specific LAN IP (*increases attack surface*)
  - Safest to define the IPv4 address to connect too (*e.g,* `10.0.0.10:60768`)
- `max_payload_len`
- `max_requests_per_min`

## Key and device storage
- `devices_file`  
  Stores the list of paired devices and their associated public identity information.
- `server_keys_file`  
  Stores the server’s cryptographic key material, including the Kyber public key and the base64-encoded ciphertext for the Kyber secret.
- `require_sealed_device_store` (recommended: `true`)  
  When enabled, NovaKey-Daemon will fail closed if the operating system's secure key storage (*keyring / sealed storage*) is unavailable.


## Logging
- `log_redact` should remain **true**
- treat logs as sensitive anyway

## Safety gates
### Arming ("push-to-type")
- `arm_enabled`
- `arm_duration_ms`
- `arm_consume_on_inject`
- `allow_clipboard_when_disarmed` (recommended: `false`)

### Two-Man approval
- `two_man_enabled`
- `approve_window_ms`
- `approve_consume_on_inject`

### Injection safety
- `allow_newlines` (recommended: `false`)
- `max_inject_len`

### Target policy
- `target_policy_enabled`
- allow/deny process names and window titles

## Recommended defaults (*most users*)
- keep `listen_addr: "0.0.0.0:60768"` unless you truly need all NICs listening
- `require_sealed_device_store: true`
- `arm_enabled: true`
- `two_man_enabled: true`
- `allow_clipboard_when_disarmed: false`

If you enable LAN listening, consider enabling target policy allowlists.
