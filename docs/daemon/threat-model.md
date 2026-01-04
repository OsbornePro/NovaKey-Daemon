---
title: Threat Model
description: Security assumptions, threats, and mitigations for NovaKey-Daemon
---

# Threat Model

This document describes the **threat model, assumptions, and mitigations** for NovaKey-Daemon.

NovaKey is designed to **safely inject secrets into local applications** using a paired remote device (e.g. phone) while minimizing the risk of misuse, exfiltration, or privilege escalation.

---

## Audience

This document is intended for:

- Security-conscious users
- Administrators exposing NovaKey on a LAN
- Auditors reviewing NovaKey’s safety properties
- Contributors working on protocol or injection logic

End users who only need basic setup may prefer the Configuration guide.

---

## Assets Protected

NovaKey aims to protect the following:

- **Secrets** (passwords, tokens, recovery keys)
- **Typing context integrity** (ensuring secrets go to the intended application)
- **User intent** (preventing unintended or automated injections)
- **Local system safety** (preventing privilege escalation or command execution)

---

## Trust Boundaries

| Component                   | Trust Level        |
|----------------------------|--------------------|
| Local NovaKey daemon        | Trusted            |
| Local OS + windowing system | Mostly trusted     |
| Paired phone                | Semi-trusted       |
| LAN                         | Untrusted          |
| Other LAN devices           | Untrusted          |
| Focused application/window  | Potentially hostile |

> “Potentially hostile” reflects the fact that NovaKey does not control the focused
> application and must assume it could be the wrong target or behave unexpectedly.

---

## Primary Attack Surfaces

### 1) Network listener (`listen_addr`)

- Accepts incoming encrypted protocol connections
- May be bound to localhost or LAN
- Exposed to packet injection, replay, or abuse if misconfigured

### 2) Paired device (phone)

- Can send arm + inject messages
- May be compromised, lost, or maliciously modified

### 3) Focused application/window

- Determines where injected text lands
- May change between arming and injection
- May be intentionally spoofed (window title tricks)

### 4) Clipboard fallback

- Can leak secrets outside intended context
- OS-level behavior varies across platforms

---

## Threat Scenarios & Mitigations

### Threat: Remote secret injection into unintended application

**Example**

- NovaKey listens on LAN
- Attacker or compromised phone injects into a terminal or admin prompt

**Mitigations**

- `target_policy_enabled`
- Allow / deny process lists
- Built-in allowlist fallback
- Two-man approval

---

### Threat: Injection into terminal or shell (command execution)

**Example**

- Secret injected into `bash`, `zsh`, `cmd.exe`
- Newlines cause command execution

**Mitigations**

- `allow_newlines: false`
- Deny terminal process names
- Two-man approval
- Arm window + consume-on-inject

---

### Threat: Race condition / focus switch attack

**Example**

- User arms NovaKey
- Focus changes before injection
- Secret lands in wrong window

**Mitigations**

- Short `arm_duration_ms`
- Target policy enforcement at injection time
- Two-man approval requires human confirmation
- Deny window title rules

---

### Threat: Replay or repeated inject attempts within arm window

**Example**

- Attacker reuses a valid arm window
- Attempts multiple injections before it expires

**Mitigations**

- Arm window timeout
- `arm_consume_on_inject: true`
- Per-device rate limiting
- Two-man approval

---

### Threat: Compromised or stolen phone

**Example**

- Attacker gains control of paired device
- Attempts repeated injections

**Mitigations**

- Arm gate (time-limited)
- Per-device rate limiting
- Target policy
- Two-man approval
- Ability to revoke device pairing

---

### Threat: LAN attacker attempts injection

**Example**

- NovaKey bound to `0.0.0.0`
- Attacker sends protocol messages

**Mitigations**

- Mutual authentication via pairing keys
- Per-device rate limiting
- Target policy
- Arm gate
- Built-in allowlist

> LAN should always be considered hostile.

---

### Threat: Clipboard exfiltration

**Example**

- Injection fails
- Secret copied to clipboard
- Clipboard manager or malware reads it

**Mitigations**

- `allow_clipboard_when_disarmed: false`
- `allow_clipboard_on_inject_failure` configurable
- Prefer injection over clipboard when possible

---

### Threat: Log leakage

**Example**

- Secrets appear in logs
- Logs written to disk or system journal

**Mitigations**

- Log redaction (`log_redact`)
- Secrets registered with redaction system
- Minimal logging of payloads

---

### Threat: Persistent credential theft from disk

**Example**

- Attacker reads `devices.json`
- Extracts long-term secrets

**Mitigations**

- Sealed device store
- `require_sealed_device_store: true`
- OS-backed encryption where available

---

## Non-Goals / Out of Scope

NovaKey does **not** attempt to protect against:

- Fully compromised local OS
- Kernel-level malware
- Keyloggers running with user privileges
- Malicious accessibility APIs
- Physical attacker with unlocked session

NovaKey assumes:

- The local user session is trusted
- The OS focus reporting is mostly accurate
- The user can visually confirm the focused application

---

## Security Design Principles

NovaKey follows these principles:

- **Fail closed** where possible
- **Short-lived authority** (arm windows, approvals)
- **Defense in depth**
- **Explicit user intent**
- **Least privilege injection**

---

## Recommended Mitigation Stack

| Threat             | Recommended Controls               |
|------------------|------------------------------------|
| LAN exposure       | Target policy + built-in allowlist |
| Focus race         | Short arm window + two-man         |
| Terminal injection | Deny terminals + no newlines       |
| Phone compromise   | Two-man + arm consume              |
| Clipboard leaks    | Disable clipboard fallback         |

---

# STRIDE Threat Analysis

This section maps NovaKey threats to the STRIDE model  
(**S**poofing, **T**ampering, **R**epudiation, **I**nformation Disclosure, **D**enial of Service, **E**levation of Privilege).

NovaKey’s design intentionally focuses on **Spoofing, Tampering, Information Disclosure, and Elevation of Privilege**, as these represent the highest-risk outcomes for secret injection systems.

---

## STRIDE Table

| STRIDE Category            | Threat Scenario                         | Impact                           | NovaKey Mitigations                                                     |
|--------------------------|------------------------------------------|----------------------------------|-------------------------------------------------------------------------|
| **Spoofing**               | Attacker pretends to be a paired device | Unauthorized injections          | Mutual pairing keys, per-device identity, rate limiting                 |
| **Spoofing**               | Window title spoofing                   | Secret injected into wrong app   | Target policy, deny rules, two-man approval                             |
| **Tampering**              | Payload modification in transit         | Corrupted or altered secrets     | Encrypted protocol, authenticated messages                              |
| **Tampering**              | Injection altered by newline execution  | Command execution                | `allow_newlines: false`, deny terminals                                 |
| **Repudiation**            | Device denies having injected secret    | Limited auditability             | Best-effort logging, device identity tracking (full non-repudiation is a non-goal) |
| **Information Disclosure** | Clipboard leaks secret                  | Secret exfiltration              | Clipboard policy flags, injection-first design                          |
| **Information Disclosure** | Logs contain secrets                    | Credential leakage               | Log redaction, secret registration                                      |
| **Information Disclosure** | Disk theft of device store              | Persistent credential compromise | Sealed device store, fail-closed option                                 |
| **Denial of Service**      | Flood of pairing or inject requests     | Resource exhaustion              | Rate limiting, pairing limits                                           |
| **Denial of Service**      | Continuous arming without inject        | User disruption                  | Arm window timeout, consume-on-inject                                   |
| **Elevation of Privilege** | Injection into terminal                 | Arbitrary command execution      | Target policy, deny rules, no newlines                                  |
| **Elevation of Privilege** | Injection into admin dialog             | Privilege escalation             | Target policy, two-man approval                                         |

---

## STRIDE Coverage Summary

| Category               | Coverage                |
|----------------------|-------------------------|
| Spoofing               | ✅ Strong               |
| Tampering              | ✅ Strong               |
| Repudiation            | ⚠️ Limited (by design) |
| Information Disclosure | ✅ Strong               |
| Denial of Service      | ⚠️ Partial             |
| Elevation of Privilege | ✅ Strong               |

> NovaKey intentionally does **not** provide non-repudiation or full audit logging.
> The system is designed for **interactive, intentional use**, not forensic reconstruction.

---

# Threat → Configuration Mapping Cheat Sheet

This section answers the question:

> “If I’m worried about **X**, which settings should I enable?”

> **Minimum viable defense**
>
> If NovaKey is listening on anything other than `127.0.0.1`, enable target policy at minimum:
>
> ```yaml
> target_policy_enabled: true
> use_built_in_allowlist: true
> ```

---

## High-Risk Threats and Recommended Configurations

### ❗ Injection into terminal / shell

**Threat:** Command execution, privilege escalation

**Config:**
```yaml
target_policy_enabled: true
denied_process_names:
  - terminal
  - bash
  - zsh
  - cmd
  - powershell
allow_newlines: false

