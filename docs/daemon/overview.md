# NovaKey-Daemon Overview

NovaKey-Daemon is a cross-platform Go agent that receives authenticated secrets
from a trusted device and injects them into the currently focused text field.

It’s designed for cases where you don’t want to type high-value secrets on your desktop keyboard:
- the secret lives on your phone
- delivery is encrypted and authenticated
- the daemon injects into the focused control (*with optional clipboard mode*)

## One port, two routes
NovaKey listens on one TCP address (`listen_addr`, default `0.0.0.0:60768`)
and routes each incoming connection by a one-line preface:

- `NOVAK/1 /pair\n` — pairing
- `NOVAK/1 /msg\n` — approve/inject messages

> **NOTE:** Message types are defined in Security → Protocol Summary (Inject/Approve/Arm/Disarm).

Clients must send a route preface line (`NOVAK/1 /msg\n` *or* `NOVAK/1 /pair\n`). Connections without a valid preface are rejected.

## Safety controls (optional)
- arming (“push-to-type”)
- two-man approval window
- injection safety rules (`allow_newlines`, `max_inject_len`)
- target policy allow/deny lists
- Arm API (*token protected*)

