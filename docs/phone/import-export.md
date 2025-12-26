# Vault Import & Export

NovaKey supports encrypted vault backups (JSON).

## Exporting a vault
1. Settings → **Export Vault**
2. Choose:
   - Protection: **None** or **Password**
   - Cipher: **AES-256-GCM** or **ChaCha20-Poly1305**
3. (Optional) Require Face ID for each secret during export
4. Save the file

## Importing a vault
1. Settings → **Import Vault**
2. Select a vault file
3. Enter password if required

Import behavior:
- existing secrets are updated
- new secrets are added
- Keychain entries are overwritten securely

## Best practices
- Store exported vaults in a secure location.
- Prefer password protection.
- Treat vault files as sensitive material.

