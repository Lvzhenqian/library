package ssh

import (
	"os"
	"testing"
)

var (
	auth = &AuthConfig{
		Username:   "root",
		Password:   "charles",
		PrivateKey: "",
		NetworkConfig: NetworkConfig{
			Network:        "tcp",
			Address:        "10.1.12.237:22",
			ConnectTimeout: 2,
		},
	}
	stdout = os.Stdout
	stderr = os.Stderr
)

func TestSSHTerminal_Run(t *testing.T) {
	cli, newClientErr := NewClient(auth)
	if newClientErr != nil {
		t.Errorf("new client error: %v", newClientErr)
		return
	}
	defer cli.Close()
	if err := cli.Run("hostname", stdout, stderr); err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_Push(t *testing.T) {
	cli, Newerr := NewClient(auth)
	if Newerr != nil {
		t.Errorf("new client error: %v", Newerr)
		return
	}
	defer cli.Close()
	if err := cli.Push("~/Downloads/jdk1.8.0_241.zip", "/tmp"); err != nil {
		t.Error(err)
	}
}

func TestSSHTerminal_Get(t *testing.T) {
	cli, Newerr := NewClient(auth)

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
	cli, Newerr := NewClient(auth)
	if Newerr != nil {
		t.Errorf("new client error: %v", Newerr)
		return
	}
	if err := cli.TunnelStart(LocalConfig, RemoteConfig); err != nil {
		t.Errorf("start ssh tunnel error: %v", err)
	}
}

func TestSshClient_Proxy(t *testing.T) {
	cli, Newerr := NewClient(auth)
	if Newerr != nil {
		t.Errorf("new client error: %v", Newerr)
		return
	}
	defer cli.Close()
	secondClient, secErr := cli.Proxy(&AuthConfig{
		Username:   "root",
		Password:   "123456",
		PrivateKey: "",
		NetworkConfig: NetworkConfig{
			Network:        "tcp",
			Address:        "10.1.12.46:22",
			ConnectTimeout: 5,
		},
	})
	if secErr != nil {
		t.Errorf("proxy second clien error: %v", secErr)
		return
	}
	defer secondClient.Close()
	secondClient.Run("hostname", stdout, stderr)
}

func ExampleNewClient() {
	cli, _ := NewClient(auth)
	cli.Run("w", stdout, stderr)
}

func ExampleSSHTerminal_Login() {
	cli, _ := NewClient(auth)
	cli.Login()
}

func ExampleSSHTerminal_Get() {
	cli, _ := NewClient(auth)
	defer cli.Close()
	cli.Get("/tmp/test02", ".")
}

func ExampleSSHTerminal_Push() {
	cli, _ := NewClient(auth)
	defer cli.Close()
	cli.Push("./test", "/tmp")
}

func ExampleSSHTerminal_TunnelStart() {
	cli, _ := NewClient(auth)
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
