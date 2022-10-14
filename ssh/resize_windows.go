//go:build windows

package ssh

// windows not support Terminal Resize
func (c *clientType) updateTerminalSize() {
	return
}
