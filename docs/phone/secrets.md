# Adding and Managing Secrets

### Create a new secret
![New secret](../assets/screenshots/novakey-new-secret.PNG)

## Add a secret
1. Tap **+**
2. Enter:
   - **Label** (example: “Email Password”)
   - **Secret**
   - **Confirm Secret**
3. Tap **Save**

### Secret actions
![Secret options](../assets/screenshots/novakey-secret-options.PNG)

## Important behavior (by design)
- Secrets are **never displayed again** after saving.
- Secrets live only in the **iOS Keychain**.
- Access requires **Face ID / passcode** when copying or sending.

## Delete a secret
- Swipe to delete, or use the secret’s action menu.

Deleting removes:
- the app record
- the Keychain entry

