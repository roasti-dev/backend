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

## Database Migrations

Migrations are managed with [goose](https://github.com/pressly/goose) and live in `internal/db/migrations/`. They run automatically on app startup.

To add a new migration, create a file following the naming convention:
```
internal/db/migrations/000012_your_migration_name.sql
```

**Migrating an existing database to goose** (run once before first deploy with goose):
```bash
scp -i ~/.ssh/roasti_deploy deploy/seed_goose_versions.sql roasti@<server-ip>:/tmp/
ssh -i ~/.ssh/roasti_deploy roasti@<server-ip> "sqlite3 /var/lib/roasti/data.db < /tmp/seed_goose_versions.sql"
```

## Deploy

Deployment is automated with Ansible.

Copy the inventory files and fill in your server details:
```bash
cp deploy/setup.example.ini deploy/setup.ini
cp deploy/deploy.example.ini deploy/deploy.ini
```

**First time setup** (installs nginx, ufw, restic, creates user and directories):
```bash
ansible-playbook -i deploy/setup.ini deploy/setup.yaml
```

After setup, obtain SSL certificate manually (only once):
```bash
sudo certbot certonly --nginx -d api.roasti.ru
```

**Deploy a new version:**
```bash
make deploy
```

### Backups

Database backups run automatically via systemd timer (4×/day) using restic to S3-compatible object storage.

**One-time backup setup:**

1. Create a bucket in your object storage
2. Fill in credentials:
```bash
cp deploy/backup.env.example deploy/backup.env
# set RESTIC_REPOSITORY, RESTIC_PASSWORD, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
```
3. Run the backup playbook:
```bash
ansible-playbook -i deploy/setup.ini deploy/backup.yaml
```

**Trigger backup manually:**
```bash
sudo systemctl start roasti-backup.service
sudo journalctl -u roasti-backup.service -n 50
```

**Restore:**
```bash
# Copy credentials to the server if needed
scp -i ~/.ssh/roasti_deploy deploy/backup.env roasti@<server-ip>:/var/lib/roasti/backup.env

# On the server:
source /var/lib/roasti/backup.env

# List available snapshots
restic snapshots

# Restore to a temp directory
restic restore latest --target /tmp/roasti-restore/

# Replace the database (stop the service first)
sudo systemctl stop roasti
sudo cp /tmp/roasti-restore/tmp/roasti-backup-*.db /var/lib/roasti/data.db
sudo systemctl start roasti
```
