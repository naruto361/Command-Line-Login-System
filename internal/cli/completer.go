package cli

import (
	"strings"
)

type commandCompleter struct {
	app *App
}

func (c *commandCompleter) Do(line []rune, pos int) ([][]rune, int) {
	var cmds []string
	if c.app.isLoggedIn() {
		cmds = c.app.commandsAfterLogin()
	} else {
		cmds = c.app.commandsBeforeLogin()
	}

	prefix := string(line[:pos])
	var suggestions [][]rune
	for _, cmd := range cmds {
		if strings.HasPrefix(cmd, prefix) {
			suggestions = append(suggestions, []rune(cmd))
		}
	}
	return suggestions, len(prefix)
}
