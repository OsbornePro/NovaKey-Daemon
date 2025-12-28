# Glossary

**Listener**  
A saved “computer target” (host/IP + port + friendly name) in the iOS app.

**Send Target**  
The Listener marked as default destination for sending secrets.

**Pairing**  
A one-time trust bootstrap that establishes mutual authentication and keying material.

**Arming**  
A local “push-to-type” gate on the daemon: injection is blocked unless armed.

**Two-Man Mode**  
A policy gate requiring an approve action before injection is allowed in a short window.

**Injection**  
Typing the secret into the currently focused text field on the computer.

**Clipboard Mode**  
When injection is blocked, the daemon may copy the secret to clipboard and report success via clipboard status.

