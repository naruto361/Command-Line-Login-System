CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE COLLATE NOCASE,
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    mfa_enabled INTEGER NOT NULL DEFAULT 0,
    totp_secret TEXT,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    account_locked INTEGER NOT NULL DEFAULT 0,
    password_reset_token TEXT,
    password_reset_expiry DATETIME,
    registered_at DATETIME NOT NULL,
    last_login DATETIME
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_reset_token ON users(password_reset_token);
