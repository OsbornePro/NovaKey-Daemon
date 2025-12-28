# Contributing to NovaKey-Daemon

Thank you for your interest in **NovaKey-Daemon**.

NovaKey-Daemon is security-critical infrastructure software.
Contributions are welcome — **but correctness, explicitness, and protocol discipline come first**.

Please read this document fully before opening an issue or submitting a pull request.

---

## Project Scope & Philosophy

**NovaKey-Daemon** is an open-source daemon implementing the NovaKey wire protocol and local injection logic.

It is designed to pair with:

* a **commercial iOS app** (NovaKey), and
* any other client that correctly implements the **published protocol**.

The daemon is:

* protocol-driven
* explicit by design
* hostile to ambiguity and silent fallback
* opinionated about security

If a behavior is not:

* explicitly encoded in the protocol, and
* reachable in code,

then it **must not exist**.

---

## What This Repository Is (and Is Not)

### ✅ This repository **is**:

* the reference implementation of the NovaKey protocol (daemon side)
* an open, reviewable security boundary
* a place for:

  * protocol fixes
  * bug fixes
  * platform support improvements
  * documentation corrections
  * security hardening

### ❌ This repository **is not**:

* the iOS app (that is proprietary)
* a UI or UX playground
* a compatibility layer for legacy clients
* a place for “best effort” security compromises

The **protocol is stable, explicit, and versioned**.
Changes must be intentional.

---

## Commercial iOS App Relationship (Important)

The NovaKey iOS app is a **commercial product** that:

* implements the **public NovaKey protocol**
* is **not open source**
* is developed in parallel but independently

Open-source contributions **do not grant rights** to:

* the iOS app source code
* branding
* App Store assets
* proprietary UX or implementation details

That said:

> **Protocol compatibility is a first-class concern.**
> If you write your own client that correctly implements the protocol, it should work.

---

## Contribution Types Welcome

### 🛠 Code Contributions

* Bug fixes
* Security hardening
* Platform-specific improvements (Linux, macOS, Windows)
* Performance improvements (measured and justified)
* Removal of dead or unreachable code

### 📚 Documentation Contributions

* Corrections to `README.md`, `PROTOCOL.md`, `SECURITY.md`
* Clarifications (must match code behavior)
* Typos and formatting improvements

### 🔍 Security Research

* Vulnerability reports
* Threat model critiques
* Cryptographic review feedback
* Protocol analysis

➡️ **Security issues must follow `SECURITY.md` — do not open public issues.**

---

## Contribution Rules (Strict but Fair)

### 1️⃣ Code Is the Source of Truth

If documentation and code disagree:

> **The code wins. Docs must be updated.**

Pull requests that modify behavior **must update documentation** where applicable.

---

### 2️⃣ No Legacy, Compatibility, or Untyped Behavior

NovaKey does **not** accept:

* legacy protocol versions
* v2 compatibility
* magic strings
* implicit routing
* untyped messages

All `/msg` behavior must map to a typed inner message:

| Type | Name    |
| ---- | ------- |
| 1    | Inject  |
| 2    | Approve |
| 3    | Arm     |
| 4    | Disarm  |

Anything else is out of scope.

---

### 3️⃣ Documentation Drift Is a Bug

This repository enforces documentation invariants.

Before submitting changes that affect behavior, contributors **must** ensure:

* `PROTOCOL.md`
* `SECURITY.md`
* user-facing docs (RTD / README)

match the code.

See `DOCS_INVARIANTS.md`.

---

### 4️⃣ Security-Relevant Behavior Must Be Explicit

If a contribution introduces or modifies:

* fallback behavior
* policy gating
* failure modes
* error conditions

Then it must:

* be explicit in code
* produce a distinct status or reason
* be documented clearly

Silent or implied behavior will be rejected.

---

### 5️⃣ Style & Testing Expectations

* Follow existing Go formatting and structure
* Avoid cleverness over clarity
* Prefer explicit branching over implicit behavior
* Add logging where it improves observability (respect redaction rules)
* Tests are encouraged where practical, especially for:

  * parsing
  * crypto boundaries
  * policy logic

---

## How to Contribute

### Issues

* Use issues for:

  * bugs
  * documentation errors
  * feature discussions (before coding)
* **Do not** use issues for security vulnerabilities

### Pull Requests

* Keep PRs focused and small
* One logical change per PR
* Include context and rationale
* Reference relevant files or protocol sections

A PR may be closed if it:

* introduces ambiguity
* weakens security posture
* adds unsupported compatibility
* diverges from protocol guarantees

---

## Security Disclosures

**Do not open public issues for security vulnerabilities.**

See `SECURITY.md` for:

* contact details
* disclosure process
* encryption options

Security researchers are welcome and appreciated.

---

## Licensing & Contributions

By contributing to this repository, you agree that:

* your contributions are licensed under the project’s open-source license
* you have the right to submit the work
* you are not contributing proprietary or confidential material

You retain copyright to your contributions.

---

## Final Notes

NovaKey-Daemon exists to be **boringly correct**.

If you value:

* explicit protocol design
* reviewable security boundaries
* long-term maintainability

you are in the right place.

Thank you for helping keep NovaKey secure.

— **Robert H. Osborne**
Maintainer, NovaKey-Daemon

