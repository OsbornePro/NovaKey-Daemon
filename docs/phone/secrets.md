# Adding and Managing Secrets

## Add a secret
1. Tap **+**
2. Enter:
   - **Label** (example: “Email Password”)
   - **Secret**
   - **Confirm Secret**
3. Tap **Save**

## Important behavior (by design)
- Secrets are **never displayed again** after saving.
- Secrets live only in the **iOS Keychain**.
- Access requires **Face ID / passcode** when copying or sending.

## Delete a secret
- Swipe to delete, or use the secret’s action menu.

Deleting removes:
- the app record
- the Keychain entry

