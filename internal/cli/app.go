package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/naruto361/command-line-login-system/internal/auth"
	"github.com/naruto361/command-line-login-system/internal/session"
	"github.com/naruto361/command-line-login-system/internal/store"
)

type App struct {
	authSvc           *auth.Service
	sess              *session.Manager
	mu                sync.Mutex
	showResetPassword bool
}

func NewApp(authSvc *auth.Service, sess *session.Manager) *App {
	return &App{authSvc: authSvc, sess: sess}
}

func (a *App) Run() {
	a.sess.SetCallbacks(a.onSessionWarn, a.onSessionExpire)

	fmt.Printf("Welcome to OSTO — secure CLI authentication\nType 'help' for available commands.\n\n")

	// readline provides tab completion and in-memory command history.
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "osto> ",
		AutoComplete:    &commandCompleter{app: a},
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: terminal not available (run with -it in Docker): %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			if err == io.EOF {
				break
			}
			fmt.Printf("Error: %v\n", err)
			break
		}

		a.executor(strings.TrimSpace(line), rl)
	}

	fmt.Println("Goodbye!")
}

func (a *App) executor(input string, rl *readline.Instance) {
	cmd := strings.TrimSpace(strings.ToLower(input))
	if cmd == "" {
		return
	}

	a.mu.Lock()
	if a.sess.CheckExpiry() {
		a.mu.Unlock()
		return
	}
	if a.sess.IsLoggedIn() && cmd != "exit" {
		a.sess.Touch()
	}
	a.mu.Unlock()

	switch cmd {
	case "help":
		a.showHelp()
	case "exit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	case "register":
		if a.isLoggedIn() {
			fmt.Println("Error: you are already logged in. Use 'logout' first.")
			return
		}
		a.handleRegister(rl)
	case "login":
		if a.isLoggedIn() {
			fmt.Println("Error: you are already logged in. Use 'logout' first.")
			return
		}
		a.handleLogin(rl)
	case "reset-password":
		if a.isLoggedIn() {
			fmt.Println("Error: you are already logged in. Use 'logout' first.")
			return
		}
		if !a.isResetPasswordAvailable() {
			fmt.Println("Error: reset-password is only available after your account is locked.")
			return
		}
		a.handleResetPassword(rl)
	case "whoami":
		if !a.isLoggedIn() {
			fmt.Println("Error: you must be logged in to use this command.")
			return
		}
		a.handleWhoami()
	case "enable-2fa":
		if !a.isLoggedIn() {
			fmt.Println("Error: you must be logged in to use this command.")
			return
		}
		if a.currentUserMFAEnabled() {
			fmt.Println("Error: MFA is already enabled. Use 'disable-2fa' to turn it off.")
			return
		}
		a.handleEnable2FA(rl)
	case "disable-2fa":
		if !a.isLoggedIn() {
			fmt.Println("Error: you must be logged in to use this command.")
			return
		}
		if !a.currentUserMFAEnabled() {
			fmt.Println("Error: MFA is not enabled. Use 'enable-2fa' to turn it on.")
			return
		}
		a.handleDisable2FA(rl)
	case "logout":
		if !a.isLoggedIn() {
			fmt.Println("Error: you are not logged in.")
			return
		}
		a.handleLogout()
	default:
		fmt.Printf("Error: unknown command '%s'. Type 'help' for available commands.\n", cmd)
	}
}

func (a *App) isLoggedIn() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sess.IsLoggedIn()
}

func (a *App) showHelp() {
	if a.isLoggedIn() {
		fmt.Println(helpAfterLoginBase)
		if a.currentUserMFAEnabled() {
			fmt.Println("  disable-2fa     Disable MFA")
		} else {
			fmt.Println("  enable-2fa      Enable Google Authenticator MFA")
		}
		return
	}
	fmt.Println(helpBeforeLoginBase)
	if a.isResetPasswordAvailable() {
		fmt.Println("  reset-password  Reset password to unlock a locked account")
	}
}

func (a *App) currentUserMFAEnabled() bool {
	s := a.sess.Current()
	if s == nil {
		return false
	}
	user, err := a.authSvc.GetUser(s.User.ID)
	return err == nil && user != nil && user.MFAEnabled
}

func (a *App) handleRegister(rl *readline.Instance) {
	fmt.Println("\n--- Register ---")
	defer restorePrompt(rl)

	var username, email, password string

	for {
		var err error
		username, err = readLine(rl, "Username (min 6 characters): ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if err := a.authSvc.CheckUsernameAvailable(username); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		break
	}

	for {
		var err error
		email, err = readLine(rl, "Email: ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if err := a.authSvc.CheckEmailAvailable(email); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		break
	}

	for {
		var err error
		password, err = readPassword(rl, "Password (min 10 chars, mixed case, digit, special): ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if err := auth.ValidatePassword(password); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		break
	}

	for {
		confirm, err := readPassword(rl, "Confirm password: ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if confirm != password {
			fmt.Printf("Error: %v\n", auth.ErrPasswordMismatch)
			continue
		}
		break
	}

	user, err := a.authSvc.Register(auth.RegisterInput{
		Username:        username,
		Email:           email,
		Password:        password,
		ConfirmPassword: password,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nSuccess: account created for '%s' (%s).\n", user.Username, user.Email)
	fmt.Println("Use 'login' to sign in.")
}

func (a *App) handleLogin(rl *readline.Instance) {
	fmt.Println("\n--- Login ---")
	defer restorePrompt(rl)

	// Username/email is collected once; only the password is re-prompted on failure.
	identifier, err := readLine(rl, "Username or email: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	user, err := a.authSvc.LookupUser(identifier)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if user == nil {
		if _, err := readPassword(rl, "Password: "); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Error: %v\n", auth.ErrInvalidCredentials)
		return
	}

	if user.AccountLocked {
		a.enableResetPassword()
		fmt.Printf("Error: %v\n", auth.ErrAccountLocked)
		fmt.Println("Use 'help' to see the reset-password command.")
		return
	}

	var authenticated *store.User
	for {
		password, err := readPassword(rl, "Password: ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		authenticated, err = a.authSvc.VerifyPassword(user, password)
		if err == nil {
			break
		}
		if errors.Is(err, auth.ErrAccountLocked) {
			a.enableResetPassword()
			fmt.Printf("Error: %v\n", err)
			fmt.Println("Use 'help' to see the reset-password command.")
			return
		}
		if errors.Is(err, auth.ErrInvalidCredentials) {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("Error: %v\n", err)
		return
	}

	user = authenticated

	if user.MFAEnabled {
		maxAttempts := a.authSvc.MaxTOTPAttempts()
		verified := false
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			code, err := readPlainLine(fmt.Sprintf("TOTP code (attempt %d/%d): ", attempt, maxAttempts))
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			if err := a.authSvc.VerifyTOTP(user, code); err != nil {
				if attempt >= maxAttempts {
					fmt.Printf("Error: %v\n", auth.ErrTOTPAttemptsExceeded)
					return
				}
				fmt.Printf("Error: %v (%d attempt(s) remaining)\n", auth.ErrInvalidTOTP, maxAttempts-attempt)
				continue
			}
			verified = true
			break
		}
		if !verified {
			return
		}
	}

	user, err = a.authSvc.CompleteLogin(user)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	a.sess.Login(user)
	a.disableResetPassword()
	fmt.Println("\nSuccess: logged in.")
	a.displayUserInfo(user)
}

func (a *App) handleResetPassword(rl *readline.Instance) {
	fmt.Println("\n--- Reset Password (unlock locked account) ---")
	defer restorePrompt(rl)

	email, err := readLine(rl, "Email: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	token, err := a.authSvc.RequestPasswordReset(email)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if token == "" {
		fmt.Println("If the email exists and the account is locked, a reset token has been sent.")
		return
	}

	fmt.Println("\nA password reset token has been sent to your email.")
	fmt.Println("(In DEV_MODE the token is also printed above.)")

	resetToken, err := readLine(rl, "Reset token: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	newPassword, err := readPassword(rl, "New password: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	confirm, err := readPassword(rl, "Confirm new password: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	err = a.authSvc.ResetPassword(auth.ResetPasswordInput{
		Email:           email,
		Token:           resetToken,
		NewPassword:     newPassword,
		ConfirmPassword: confirm,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\nSuccess: password reset. Your account is unlocked. Use 'login' to sign in.")
	a.disableResetPassword()
}

func (a *App) handleWhoami() {
	s := a.sess.Current()
	if s == nil {
		fmt.Println("Error: session expired. Please log in again.")
		return
	}

	user, err := a.authSvc.GetUser(s.User.ID)
	if err != nil || user == nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	a.displayUserInfo(user)
}

func (a *App) enableResetPassword() {
	a.mu.Lock()
	a.showResetPassword = true
	a.mu.Unlock()
}

func (a *App) disableResetPassword() {
	a.mu.Lock()
	a.showResetPassword = false
	a.mu.Unlock()
}

func (a *App) isResetPasswordAvailable() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.showResetPassword
}

func (a *App) handleEnable2FA(rl *readline.Instance) {
	s := a.sess.Current()
	if s == nil {
		fmt.Println("Error: session expired. Please log in again.")
		return
	}

	result, err := a.authSvc.EnableMFA(s.User.ID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\nScan this QR code with Google Authenticator:")
	printQRCode(result.URL)
	fmt.Printf("\nManual entry secret: %s\n", result.Secret)

	defer restorePrompt(rl)
	code, err := readPlainLine("Enter the 6-digit code from your authenticator to confirm: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if err := a.authSvc.ConfirmEnableMFA(s.User.ID, result.Secret, code); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("MFA was not enabled. Try 'enable-2fa' again.")
		return
	}

	fmt.Println("\nSuccess: MFA enabled.")

	user, _ := a.authSvc.GetUser(s.User.ID)
	if user != nil {
		a.sess.RefreshUser(user)
	}
}

func (a *App) handleDisable2FA(rl *readline.Instance) {
	s := a.sess.Current()
	if s == nil {
		fmt.Println("Error: session expired. Please log in again.")
		return
	}

	fmt.Println("\n--- Disable MFA ---")
	defer restorePrompt(rl)

	password, err := readPassword(rl, "Current password: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	code, err := readPlainLine("TOTP code: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if err := a.authSvc.DisableMFA(s.User.ID, password, code); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\nSuccess: MFA disabled.")
	user, _ := a.authSvc.GetUser(s.User.ID)
	if user != nil {
		a.sess.RefreshUser(user)
	}
}

func (a *App) handleLogout() {
	a.sess.Logout()
	fmt.Println("\nSuccess: logged out.")
}

func (a *App) displayUserInfo(user *store.User) {
	fmt.Println("\n--- User Details ---")
	fmt.Printf("Username:          %s\n", user.Username)
	fmt.Printf("Email:             %s\n", user.Email)
	fmt.Printf("Registration date: %s\n", user.RegisteredAt.UTC().Format(time.RFC3339))
	if user.MFAEnabled {
		fmt.Println("MFA status:        enabled")
	} else {
		fmt.Println("MFA status:        disabled")
	}
	if user.LastLogin != nil {
		fmt.Printf("Last login:        %s\n", user.LastLogin.UTC().Format(time.RFC3339))
	} else {
		fmt.Println("Last login:        N/A")
	}
	if s := a.sess.Current(); s != nil {
		fmt.Printf("Session expires at: %s\n", s.ExpiresAt.UTC().Format(time.RFC3339))
	}
}

func (a *App) onSessionWarn(expiresAt time.Time) {
	fmt.Printf("\n[WARNING] Your session will expire at %s (in less than 5 minutes). Activity will extend your session.\n",
		expiresAt.UTC().Format(time.RFC3339))
}

func (a *App) onSessionExpire() {
	fmt.Println("\n[SESSION EXPIRED] You have been automatically logged out.")
}

func printQRCode(content string) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		fmt.Println("(Could not generate QR code; use the manual secret instead.)")
		return
	}
	fmt.Println(qr.ToSmallString(false))
}
