//go:build !windows

package ssh

import (
	terminal "golang.org/x/term"
	"os"
	"os/signal"
	"syscall"
)

func (c *ClientType) updateTerminalSize(fd, termWidth, termHeight int, failed chan error) {
	sigwinchCh := make(chan os.Signal, 1)
	signal.Notify(sigwinchCh, syscall.SIGWINCH)

	for range sigwinchCh {
		currTermWidth, currTermHeight, getSizeErr := terminal.GetSize(fd)
		if getSizeErr != nil {
			failed <- getSizeErr
			continue
		}
		if currTermHeight == termHeight && currTermWidth == termWidth {
			continue
		}

		if changeErr := c.session.WindowChange(currTermHeight, currTermWidth); changeErr != nil {
			failed <- changeErr
			continue
		}
		termWidth, termHeight = currTermWidth, currTermHeight
	}
}
