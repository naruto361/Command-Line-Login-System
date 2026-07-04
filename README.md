# OSTO — Secure CLI Authentication

A Dockerized Go CLI application with SQLite persistence, bcrypt password hashing, account lockout, email-based password reset, optional Google Authenticator (TOTP) MFA, and session management.

## Features

- **Registration** — username (min 6 chars), email, password (min 10 chars with upper, lower, digit, special)
- **Login** — username or email + password; optional TOTP step when MFA is enabled
- **Account lockout** — configurable failed attempt limit (default 5); unlock via password reset
- **Password reset** — email token flow for locked accounts
- **MFA** — enable/disable Google Authenticator (QR code + manual secret)
- **Session** — configurable timeout (default 30 min), 5-minute expiry warning, auto-logout
- **Interactive CLI** — tab completion, in-memory history, masked password input

## Prerequisites

- [Go](https://go.dev/dl/) 1.22+
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose

## Quick Start (Docker)

```bash
# Build and start the interactive CLI
docker compose run --rm osto

# Or using Make
make docker-run
```

The SQLite database is stored at `/app/data/app.db` inside a **named Docker volume** (`osto_data`), so data persists across container restarts.

## Local Development

```bash
# Install dependencies
go mod tidy

# Run tests
make test

# Build and run locally (uses ./data/app.db)
set DATABASE_PATH=./data/app.db
make run
```

## Configuration

All settings are driven by environment variables (see `.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_PATH` | `/app/data/app.db` | SQLite database file path |
| `MAX_FAILED_LOGIN_ATTEMPTS` | `5` | Failed logins before account lock |
| `MAX_TOTP_ATTEMPTS` | `3` | TOTP retries per login attempt |
| `SESSION_TIMEOUT_MINUTES` | `30` | Session duration |
| `SESSION_WARNING_MINUTES` | `5` | Warn before session expiry |
| `PASSWORD_RESET_TOKEN_EXPIRY_MINUTES` | `60` | Reset token lifetime |
| `DEV_MODE` | `true` | Print reset tokens to console (no SMTP) |
| `APP_NAME` | `OSTO` | Shown in TOTP issuer / emails |

## CLI Commands

### Before login

| Command | Description |
|---------|-------------|
| `register` | Create a new account |
| `login` | Sign in with username/email and password |
| `reset-password` | Reset password to unlock a locked account |
| `help` | List available commands |
| `exit` | Quit |

### After login

| Command | Description |
|---------|-------------|
| `whoami` | Show user details and session expiry |
| `enable-2fa` | Enable Google Authenticator MFA |
| `disable-2fa` | Disable MFA (requires password + TOTP) |
| `logout` | End session |
| `help` | List available commands |
| `exit` | Quit |

## Example Session

```
osto> register
Username (min 6 characters): alice1
Email: alice@example.com
Password: **********
Confirm password: **********
Success: account created for 'alice1' (alice@example.com).

osto> login
Username or email: alice1
Password: **********
Success: logged in.

--- User Details ---
Username:          alice1
Email:             alice@example.com
Registration date: 2026-07-04T10:00:00Z
MFA status:        disabled
Last login:        2026-07-04T10:05:00Z
Session expires:   2026-07-04T10:35:00Z

osto> enable-2fa
Success: MFA enabled.
(scan QR code or use manual secret)

osto> logout
Success: logged out.
```

## Project Structure

```
.
├── cmd/cli/main.go           # Application entrypoint
├── internal/
│   ├── auth/                 # Registration, login, lockout, reset
│   ├── cli/                  # Interactive REPL and commands
│   ├── config/               # Environment-based configuration
│   ├── email/                # SMTP / dev-mode email sender
│   ├── mfa/                  # TOTP generation and validation
│   ├── session/              # In-memory session management
│   └── store/                # SQLite repository + schema
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Security Notes

- Passwords are hashed with **bcrypt** (cost 12); plaintext passwords are never stored.
- TOTP secrets are stored in plain text (acceptable for this assignment; production should encrypt at rest).
- Account lockout requires **password reset** — there is no timed auto-unlock.
- Failed login attempts reset to zero on successful login.
- Password reset tokens are cryptographically random and time-limited.
- Generic error messages prevent user enumeration where practical.
- No secrets, database files, or `.env` files are committed to the repository.

## Make Targets

```bash
make build        # Build binary to bin/osto
make test         # Run all unit tests
make docker-build # Build Docker image
make docker-run   # Run interactive CLI in Docker
make docker-down  # Stop and remove containers
make tidy         # go mod tidy
```

## License

MIT
