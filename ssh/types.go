package ssh

import (
	"io"
)

// same as net.Dial
type NetworkConfig struct {
	Network string
	// 如果Network 为unix，则Address为对应的文件路径
	// 如果Network 为tcp，则Address为 ip:port
	Address        string
	ConnectTimeout int
}

type AuthConfig struct {
	Username   string
	Password   string
	PrivateKey string
	NetworkConfig
}

type Client interface {
	Login() error
	Run(cmd string, output io.Writer) error
	Get(src, dst string) error
	Push(src, dst string) error
	TunnelStart(Local, Remote NetworkConfig) error
	Proxy(RemoteAuthConfig *AuthConfig) (Client, error)
	Close() error
}
