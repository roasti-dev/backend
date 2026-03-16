# Roasti app

## Setting up a local development environment

### Start the Firebase emulator:
```bash
make firebase-emulator
export FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099
export FIREBASE_IDENTITY_BASE_URL=http://localhost:9099/identitytoolkit.googleapis.com/v1/accounts
export FIREBASE_TOKEN_BASE_URL=http://localhost:9099/securetoken.googleapis.com/v1/token
```

### Start the app:
```bash
SERVER_PORT=9090 DEBUG=1 go run ./cmd/server
```

Swagger documentation is available at `http://localhost:9090/docs`

## Linting
```bash
make lint
```

## E2E Testing

Make sure the app and Firebase emulator are running before executing e2e tests.
```bash
make e2e
```

## Deploy

Deployment is automated with Ansible.

Copy the inventory file and fill in your server details:
```bash
cp deploy/inventory.example.ini deploy/inventory.ini
```

First time setup (creates user, directories, and systemd service):
```bash
make setup-server
```

Deploy a new version:
```bash
make deploy
```