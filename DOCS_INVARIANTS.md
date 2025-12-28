# 🧭 Docs ↔ Protocol Invariants (Drift Prevention Rule)

## 📐 Documentation–Protocol Invariants

To prevent documentation drift in security-critical areas, NovaKey follows these rules:

---

### 1. **Docs must describe only accepted wire behavior**

If the daemon rejects it, the docs must not mention it.

**Examples:**

* ❌ No “legacy inject”
* ❌ No “implicit routing”
* ❌ No “magic strings”
* ❌ No untyped messages

If it’s not reachable in code, it must not appear in docs.

Any documentation change that describes wire or protocol behavior **must be reviewed against the current daemon code before release**.

---

### 2. **Every `/msg` maps to an explicit inner message type**

All message behavior described in documentation must reference one of:

* Inject (**1**)
* Approve (**2**)
* Arm (**3**) — protocol message, not the local HTTP Arm API
* Disarm (**4**) — protocol message, not the local HTTP Arm API

If a behavior cannot be expressed as a typed inner message, it is **not a supported protocol feature**.

---

### 3. **Security-relevant fallbacks must be explicit and observable**

If a fallback exists (*e.g., clipboard*), documentation must state:

* when it can occur
* that it is policy-gated
* that it produces a distinct result/status

Silent, implied, or ambiguous fallback language is forbidden.

---

### 4. **Protocol version statements are authoritative**

Docs must never imply support for:

* v2
* pre-inner framing
* compatibility modes
* untyped messages

The only supported `/msg` protocol is **Protocol v3 with Inner Frame v1**.

---

### 5. **Code is the source of truth**

When docs and code disagree:

> **The code is correct. Docs must be updated.**

Security reviews and releases must validate that:

* `PROTOCOL.md`
* `SECURITY.md`
* RTD summaries

all reflect the current code paths.

---

## 🔍 Verify No Legacy Terminology

The following command **must produce no output**:

```bash
rg -n '(legacy|v2|compat|pre-inner|magic string|implicit|untyped)' docs/ README.md && exit 1
```

