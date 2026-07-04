package cli

const helpBeforeLoginBase = `Available commands (not logged in):
  register        Create a new user account
  login           Log in with username/email and password
  help            Show this help message
  exit            Quit the application`

const helpAfterLoginBase = `Available commands (logged in):
  whoami          Show current user details
  logout          End the current session
  help            Show this help message
  exit            Quit the application`

func (a *App) commandsBeforeLogin() []string {
	cmds := []string{"register", "login", "help", "exit"}
	if a.isResetPasswordAvailable() {
		cmds = []string{"register", "login", "reset-password", "help", "exit"}
	}
	return cmds
}

func (a *App) commandsAfterLogin() []string {
	cmds := []string{"whoami", "logout", "help", "exit"}
	if a.currentUserMFAEnabled() {
		cmds = []string{"whoami", "disable-2fa", "logout", "help", "exit"}
	} else {
		cmds = []string{"whoami", "enable-2fa", "logout", "help", "exit"}
	}
	return cmds
}
