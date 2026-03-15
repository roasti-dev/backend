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

## Systemd service configuration

Create a service file at `/etc/systemd/system/<service-name>.service`:
```ini
[Unit]
Description=Backend Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/home/app
ExecStart=/home/app/app-debian
Restart=always
RestartSec=3
Environment=APP_ENV=production

[Install]
WantedBy=multi-user.target
```

Then enable and start the service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable <service-name>
sudo systemctl start <service-name>
```
