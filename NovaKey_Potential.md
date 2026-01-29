# NOVAK Protocol Potential

> *NovaKey was built to securely deliver secrets for typing.  
> The protocol underneath it turned out to be something else.*

## Background

NovaKey uses the **NOVAK wire protocol** to deliver high-trust, authenticated messages from a client device to a listener. In the current implementation, this is used to inject secrets (passwords, tokens, etc.) into focused fields in a controlled and auditable way.

However, after building and using it, it’s become clear that **secret delivery is just one narrow application** of what the protocol actually enables.

At its core, NOVAK is not a “password protocol.”  
It is a **human-mediated, cryptographically enforced interaction protocol**.

This document outlines some of the broader directions NOVAK could be taken, in case someone smarter than me wants to run with the idea.

---

## What NOVAK Actually Is

The protocol has a few defining characteristics:

- **Strong identity**  
  Devices are explicitly paired and authenticated. There is no ambient or anonymous access.

- **Intent-scoped messages**  
  Messages are typed (`Inject`, `Approve`, `Arm`, `Disarm`, etc.), not generic data blobs.

- **One intent per connection**  
  Each connection represents a single, explicit action or authorization.

- **Human-in-the-loop by design**  
  The protocol assumes that a human is involved, present, and making a decision.

- **Short-lived authority**  
  Actions can be gated by time windows, approvals, or explicit arming.

- **Meaningful auditability**  
  Messages describe *what was intended*, not just *what bytes were sent*.

This puts NOVAK much closer to **secure action authorization** than to traditional transport or monitoring protocols.

---

## The Core Idea: Secure Human Interaction

Most infrastructure protocols secure *transport*:
- TLS secures pipes
- API tokens secure endpoints
- SNMPv3 secures PDUs

NOVAK secures **intent**.

It answers questions like:
- *Who authorized this?*
- *What exactly were they authorizing?*
- *Was a human present?*
- *Was this allowed right now, under these conditions?*

That turns out to be useful in far more places than secret injection.

---

## Potential Applications Beyond Secret Delivery

Below are examples of areas where NOVAK’s properties make sense. This is not an exhaustive list.

### 1. Human-Gated Control Plane Actions

- Approving deployments
- Restarting services
- Enabling maintenance mode
- Promoting canaries
- Pausing automation

Instead of long-lived API tokens or SSH access, actions are explicitly authorized and narrowly scoped.

---

### 2. Break-Glass / Emergency Operations

- Disabling authentication globally
- Revoking all sessions
- Forcing key rotation
- Locking down infrastructure

These actions are rare, high-impact, and should feel deliberate. NOVAK naturally enforces friction and intent.

---

### 3. Secure Event Handler Execution

Systems like monitoring platforms or schedulers could trigger **predefined, whitelisted handlers**, but only when explicitly authorized via NOVAK.

This avoids turning remote execution into a generic shell while still enabling controlled remediation.

---

### 4. Approval Gates for Automation & CI/CD

Pipelines often need a “yes” from a human:
- before deploying to production
- before deleting data
- before performing irreversible actions

NOVAK can act as a cryptographic approval oracle rather than another webhook or API token.

---

### 5. Secret *Release* Authorization

Instead of delivering secrets directly:
- NOVAK authorizes the release or use of a secret
- The secret never transits the protocol
- Access is time-bound, scoped, and auditable

This pairs well with external vaults or HSMs.

---

### 6. Operator Presence Signaling

Some systems need to know:
> “Is a human actively paying attention?”

NOVAK can assert human presence, acknowledgment, or intent in a way that automation can reason about safely.

---

### 7. Replacing Ad-Hoc SSH for One-Off Actions

Many production incidents end with:
> “I SSH’d in and ran a command.”

NOVAK offers a safer alternative:
- no shell
- no shared keys
- no ambient authority
- explicit, logged intent

---

### 8. Physical or Hybrid Systems

Because NOVAK actions are rare and deliberate, it could bridge into:
- access control
- hardware interlocks
- lab equipment
- power or network toggles

Anywhere that *doing the wrong thing casually* is unacceptable.

---

## What NOVAK Is Probably Not

NOVAK is **not** well suited for:
- high-volume telemetry
- streaming metrics
- bulk data transfer
- generic RPC replacement

It shines when **humans matter** and **mistakes are expensive**.

---

## Why This Is Interesting

Very few protocols are designed around the idea that:
> “A human explicitly authorized this specific action.”

Most systems approximate this with tokens, sessions, and roles.

NOVAK makes it first-class.

---

## Closing

NovaKey uses NOVAK today to deliver secrets for typing.

The protocol underneath it appears to be more generally useful as a **secure, intent-driven interaction layer between humans and systems**.

This document exists to capture that realization and invite exploration.

If you see something here worth extending, formalizing, or completely rethinking — please do.