# Glossary

**Listener**  
A paired computer running NovaKey-Daemon.
A saved “computer target” (host/IP + port + friendly name) in the iOS app.

**Secret**  
Sensitive data stored securely in the NovaKey app.

**Pairing**  
The process of establishing cryptographic trust between devices.
A one-time trust bootstrap that establishes mutual authentication and keying material.

**Pro Unlock**  
A one-time purchase that removes Free-tier limits.

**Send Target**  
The Listener marked as default destination for sending secrets.

**Arming**  
A local “push-to-type” gate on the daemon: injection is blocked unless armed.

**Two-Man Mode**  
A policy gate requiring an approve action before injection is allowed in a short window.

**Injection**  
Typing the secret into the currently focused text field on the computer.

**Clipboard Mode**  
When injection is blocked, the daemon may copy the secret to clipboard and report success via clipboard status.

