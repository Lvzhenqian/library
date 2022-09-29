// +build windows

package sshtool

// windows not support Terminal Resize
func (c *SshClient) updateTerminalSize() {
	return
}
