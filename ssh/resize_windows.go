//go:build windows

package ssh

import (
	terminal "golang.org/x/term"
	"time"
)

// windows not support Terminal Resize
func (c *ClientType) updateTerminalSize(fd, termWidth, termHeight int, failed chan error) {
	for range time.Tick(time.Second) {
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
