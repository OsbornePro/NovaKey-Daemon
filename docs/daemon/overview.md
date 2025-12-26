# NovaKey-Daemon Overview

NovaKey-Daemon is a cross-platform Go agent that receives authenticated secrets
from a trusted device and injects them into the currently focused text field.

It’s designed for cases where you don’t want to type high-value secrets on your desktop keyboard:
- the secret lives on your phone
- delivery is encrypted and authenticated
- the daemon injects into the focused control (with optional clipboard fallback)

## One port, two routes
NovaKey listens on one TCP address (`listen_addr`, default `127.0.0.1:60768`)
and routes each incoming connection by a one-line preface:

- `NOVAK/1 /pair\n` — pairing
- `NOVAK/1 /msg\n` — approve/inject messages

If the route line is absent, the daemon treats it as `/msg` for compatibility.

## Safety controls (optional)
- arming (“push-to-type”)
- two-man approval window
- injection safety rules (`allow_newlines`, `max_inject_len`)
- target policy allow/deny lists
- local Arm API (loopback only, token protected)

