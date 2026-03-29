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

## Seeding data
 
To populate the database with initial data for local development:
```bash
DATABASE_PATH=data.db go run ./cmd/seed --user=test --type recipes --file ./cmd/seed/data/recipes.json
```

## Linting
```bash
make lint
```

## E2E Testing

```bash
make test-e2e
```

## Deploy

Deployment is automated with Ansible.

Copy the inventory file and fill in your server details:
```bash
cp deploy/inventory.example.ini deploy/inventory.ini
```

First time setup (installs nginx, ufw, creates user and directories):
```bash
make setup-server
```

After setup, obtain SSL certificate manually (only once):
```bash
sudo certbot certonly --nginx -d api.roasti.ru
```

Deploy a new version:
```bash
make deploy
