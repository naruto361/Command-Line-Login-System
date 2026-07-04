package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

func readLine(rl *readline.Instance, prompt string) (string, error) {
	rl.SetPrompt(prompt)
	line, err := rl.Readline()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// readLineAfterOutput resets readline after multi-line prints (e.g. QR codes)
// so keystrokes don't insert extra blank lines.
func readLineAfterOutput(rl *readline.Instance, prompt string) (string, error) {
	rl.Clean()
	rl.Refresh()
	return readLine(rl, prompt)
}

// readPlainLine uses simple stdin for short inputs after heavy terminal output.
func readPlainLine(prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func readPassword(rl *readline.Instance, prompt string) (string, error) {
	bytes, err := rl.ReadPassword(prompt)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func promptCredentials(rl *readline.Instance) (identifier, password string, err error) {
	identifier, err = readLine(rl, "Username or email: ")
	if err != nil {
		return "", "", err
	}
	password, err = readPassword(rl, "Password: ")
	if err != nil {
		return "", "", err
	}
	return identifier, password, nil
}

func restorePrompt(rl *readline.Instance) {
	rl.SetPrompt("osto> ")
	rl.Clean()
}
