package sshtool

import (
	"os"
	"testing"
)

var authConfig = &AuthConfig{
	Username:   "root",
	Password:   "charles",
	PrivateKey: "",
	NetworkConfig: NetworkConfig{
		Network:        "tcp",
		Address:        "10.1.12.237:22",
		ConnectTimeout: 2,
	},
}

func TestSSHTerminal_Run(t *testing.T) {
	cli, newClientErr := NewClient(authConfig)
	if newClientErr != nil {
		t.Errorf("new client error: %v", newClientErr)
		return
	}
	defer cli.Close()
	if err := cli.Run("hostname", os.Stdout); err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_Push(t *testing.T) {
	cli, Newerr := NewClient(authConfig)
	if Newerr != nil {
		t.Errorf("new client error: %v", Newerr)
		return
	}
	defer cli.Close()
	if err := cli.Push("~/Downloads/log.sh", "/tmp"); err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_Get(t *testing.T) {
	cli, Newerr := NewClient(authConfig)

	if Newerr != nil {
		t.Errorf("new client error: %v", Newerr)
		return
	}
	defer cli.Close()
	if err := cli.Get("/tmp/a.txt", "/Users/charles/Downloads"); err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_TunnelStart(t *testing.T) {
	LocalConfig := NetworkConfig{
		Network: "tcp",
		Address: "127.0.0.1:6666",
	}
	RemoteConfig := NetworkConfig{
		Network: "tcp",
		Address: "10.1.12.46:22",
	}
	cli, Newerr := NewClient(authConfig)
	if Newerr != nil {
		t.Errorf("new client error: %v", Newerr)
		return
	}
	if err := cli.TunnelStart(LocalConfig, RemoteConfig);err != nil {
		t.Errorf("start ssh tunnel error: %v",err)
	}
}

func TestSshClient_Proxy(t *testing.T) {
	cli, Newerr := NewClient(authConfig)
	remoteAuth := AuthConfig{
		Username:      "root",
		Password:      "123456",
		PrivateKey:    "",
		NetworkConfig: NetworkConfig{
			Network: "tcp",
			Address: "10.1.12.46:22",
			ConnectTimeout: 5,
		},
	}
	if Newerr != nil {
		t.Errorf("new client error: %v", Newerr)
		return
	}
	defer cli.Close()
	secondClient ,secErr:= cli.Proxy(remoteAuth)
	if secErr != nil {
		t.Errorf("proxy second clien error: %v",secErr)
		return
	}
	defer  secondClient.Close()
	secondClient.Run("hostname",os.Stdout)
}

func ExampleNewClient() {
	cli, _ := NewClient(authConfig)
	cli.Run("w", os.Stdout)
}

func ExampleSSHTerminal_Login() {
	cli, _ := NewClient(authConfig)
	cli.Login()
}

func ExampleSSHTerminal_Get() {
	cli, _ := NewClient(authConfig)
	defer cli.Close()
	cli.Get("/tmp/test02", ".")
}

func ExampleSSHTerminal_Push() {
	cli, _ := NewClient(authConfig)
	defer cli.Close()
	cli.Push("./test", "/tmp")
}

func ExampleSSHTerminal_TunnelStart() {
	cli, _ := NewClient(authConfig)
	local := NetworkConfig{
		Network: "tcp",
		Address: "127.0.0.1:9000",
	}
	remote := NetworkConfig{
		Network: "unix",
		Address: "/var/run/docker.sock",
	}
	cli.TunnelStart(local, remote)
}
