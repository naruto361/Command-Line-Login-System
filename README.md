# OSTO — Secure CLI Authentication

A Dockerized Go CLI application with SQLite persistence, bcrypt password hashing, account lockout, email-based password reset, optional Google Authenticator (TOTP) MFA, and session management.

**Repository:** [github.com/naruto361/Command-Line-Login-System](https://github.com/naruto361/Command-Line-Login-System)

## Features

- **Registration** — username (min 6 chars), email, password (min 10 chars with upper, lower, digit, special)
- **Login** — username or email + password; optional TOTP step when MFA is enabled
- **Account lockout** — configurable failed attempt limit (default 5); unlock via password reset
- **Password reset** — email token flow for locked accounts (token printed in DEV_MODE)
- **MFA** — enable/disable Google Authenticator (QR code + manual secret)
- **Session** — configurable timeout (default 30 min), 5-minute expiry warning, auto-logout
- **Interactive CLI** — tab completion, in-memory history, masked password input

## Prerequisites

- [Go](https://go.dev/dl/) 1.22+
- [Docker Desktop](https://docs.docker.com/get-docker/) & Docker Compose (for containerized run)

---

## Quick Start — Clone & Run

### 1. Clone the repository

```bash
git clone https://github.com/naruto361/Command-Line-Login-System.git
cd Command-Line-Login-System
```

### 2. Run with Docker (recommended for submission demo)

Start **Docker Desktop**, then in a terminal (PowerShell, Windows Terminal, or bash):

```bash
docker compose build
docker compose run --rm -it osto
```

> **Important:** Use the `-it` flags so the interactive CLI stays open and accepts input.

The SQLite database is stored at `/app/data/app.db` inside a **named Docker volume** (`osto_data`), so data persists across container restarts. No database file is committed to the repository — tables are created automatically on first run.

### 3. Run locally with Go (alternative)

**Windows (PowerShell):**

```powershell
go mod tidy
$env:DATABASE_PATH = ".\data\app.db"
go run .\cmd\cli
```

**Mac / Linux:**

```bash
go mod tidy
export DATABASE_PATH=./data/app.db
go run ./cmd/cli
```

You should see:

```text
Welcome to OSTO — secure CLI authentication
osto>
```

---

## Example test flow

```text
osto> register
# Username: alice1
# Email: alice@example.com
# Password: SecurePass1!

osto> login
# Username or email: alice1
# Password: SecurePass1!

osto> whoami
osto> enable-2fa
# Scan QR code, enter 6-digit TOTP code

osto> logout
osto> login
# Enter password + TOTP code

osto> help
osto> exit
```

---

## Configuration

All settings are driven by environment variables in **`docker-compose.yml`** (see also `.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_PATH` | `/app/data/app.db` | SQLite database file path |
| `MAX_FAILED_LOGIN_ATTEMPTS` | `5` | Failed logins before account lock |
| `MAX_TOTP_ATTEMPTS` | `3` | TOTP retries per login attempt |
| `SESSION_TIMEOUT_MINUTES` | `30` | Session duration |
| `SESSION_WARNING_MINUTES` | `5` | Warn before session expiry |
| `PASSWORD_RESET_TOKEN_EXPIRY_MINUTES` | `60` | Reset token lifetime |
| `DEV_MODE` | `true` | Print reset tokens to console (no SMTP needed) |
| `APP_NAME` | `OSTO` | Shown in TOTP issuer / emails |

Config-only changes do **not** require a Docker rebuild — restart the container:

```bash
docker compose run --rm -it osto
```

Rebuild is required only when **Go source code** or the **Dockerfile** changes:

```bash
docker compose build
```

---

## CLI Commands

### Before login

| Command | Description |
|---------|-------------|
| `register` | Create a new account |
| `login` | Sign in with username/email and password (+ TOTP if MFA enabled) |
| `reset-password` | Unlock a locked account *(appears in help only after lockout)* |
| `help` | List available commands |
| `exit` | Quit |

### After login

| Command | Description |
|---------|-------------|
| `whoami` | Show current user details and session expiry |
| `enable-2fa` | Enable Google Authenticator MFA *(shown only when MFA is disabled)* |
| `disable-2fa` | Disable MFA *(shown only when MFA is enabled; requires password + TOTP)* |
| `logout` | End session |
| `help` | List available commands |
| `exit` | Quit |

After a successful login, user details are displayed automatically:

- Username
- Registration date
- MFA status (enabled/disabled)
- Last login time
- Session expiration time

---

## Run tests

```bash
go test ./... -count=1
```

Or with Make (Git Bash / Mac / Linux):

```bash
make test
```

---

## Project Structure

```
.
├── cmd/cli/main.go           # Application entrypoint
├── internal/
│   ├── auth/                 # Registration, login, lockout, password reset
│   ├── cli/                  # Interactive REPL and commands
│   ├── config/               # Environment-based configuration
│   ├── email/                # SMTP / dev-mode email sender
│   ├── mfa/                  # TOTP generation and validation
│   ├── session/              # In-memory session management
│   └── store/                # SQLite repository + schema.sql
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## Security Notes

- Passwords are hashed with **bcrypt** (cost 12); plaintext passwords are never stored.
- TOTP secrets are stored in plain text (acceptable for this assignment; production should encrypt at rest).
- Account lockout requires **password reset** — there is no timed auto-unlock.
- Failed login attempts reset to zero on successful login.
- Password reset tokens are cryptographically random and time-limited.
- Generic error messages reduce user enumeration where practical.
- No secrets, database files (`.db`), or `.env` files are committed to the repository.

---

## Make Targets

```bash
make build        # Build binary to bin/osto
make test         # Run all unit tests
make docker-build # Build Docker image
make docker-run   # Run interactive CLI in Docker (-it)
make docker-down  # Stop and remove containers
make tidy         # go mod tidy
```

---

## Submission

This project fulfills the **Containerized CLI Login System with Optional 2FA** assignment:

- Go CLI with registration, login, MFA, lockout, and session management
- Docker + SQLite with persistent named volume
- Unit tests for core authentication flows
- Public repository: **https://github.com/naruto361/Command-Line-Login-System**

## License

MIT
