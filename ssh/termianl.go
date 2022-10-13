package sshtool

import (
	"bufio"
	"fmt"
	"github.com/kr/fs"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SshClient struct {
	session *ssh.Session
	client  *ssh.Client
	exitMsg string
	stdout  io.Reader
	stdin   io.Writer
	stderr  io.Reader
}

func NewClient(config ClientConfig) (Client, error) {
	var (
		auth []ssh.AuthMethod
		ac   = config.Config()
	)
	if ac.Password != "" {
		auth = []ssh.AuthMethod{ssh.Password(ac.Password)}
	} else {
		if ac.PrivateKey == "" {
			ac.PrivateKey = "~/.ssh/id_rsa"
		}
		var (
			content []byte
			readErr error
		)
		_, err := os.Stat(ac.PrivateKey)
		if os.IsExist(err) {
			content, readErr = os.ReadFile(ac.PrivateKey)
			if readErr != nil {
				return nil, fmt.Errorf("open private key %s error: %w", ac.PrivateKey, readErr)
			}
		} else {
			content = []byte(ac.PrivateKey)
		}
		signer, parseErr := ssh.ParsePrivateKey(content)
		if parseErr != nil {
			return nil, fmt.Errorf("parse private key %s error: %w", ac.PrivateKey, parseErr)
		}
		auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}

	cfg := &ssh.ClientConfig{
		User:            ac.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(ac.ConnectTimeout) * time.Second,
	}
	cli, err := ssh.Dial(ac.Network, ac.Address, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect error: %w", err)
	}
	session, getSessionErr := cli.NewSession()
	if getSessionErr != nil {
		return nil, getSessionErr
	}
	return &SshClient{client: cli, session: session}, nil
}

func TotalSize(paths string) int64 {
	var Ret int64
	stat, _ := os.Stat(paths)
	switch {
	case stat.IsDir():
		filepath.Walk(paths, func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			} else {
				s, _ := os.Stat(p)
				Ret = Ret + s.Size()
				return nil
			}
		})
		return Ret
	default:
		return stat.Size()
	}
}

func LocalRealPath(ph string) string {
	sl := strings.Split(ph, "/")
	if sl[0] == "~" {
		s, ok := os.LookupEnv("HOME")
		if !ok {
			panic("Get $HOME Env Error!!")
		}
		sl[0] = s
		return path.Join(sl...)
	}
	return ph
}

func RemoteRealpath(ph string, c *sftp.Client) string {
	sl := strings.Split(ph, "/")
	if sl[0] == "~" {
		r, e := c.Getwd()
		if e != nil {
			panic("Get Remote $HOME Error!!")
		}
		sl[0] = r
		return path.Join(sl...)
	}
	return ph
}

func testPort(n NetworkConfig) bool {
	_, err := net.DialTimeout(n.Network, n.Address, time.Duration(n.ConnectTimeout)*time.Second)
	if err != nil {
		return false
	}
	return true
}

func (c *SshClient) closeSession() error {
	return c.session.Close()
}

func (c *SshClient) interactiveSession() error {
	defer c.closeSession()
	defer func() {
		if c.exitMsg == "" {
			fmt.Fprintln(os.Stdout, "the connection was closed on the remote side on ", time.Now().Format(time.RFC822))
		} else {
			fmt.Fprintln(os.Stdout, c.exitMsg)
		}
	}()

	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer terminal.Restore(fd, state)

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		return err
	}

	termType, ok := os.LookupEnv("TERM")

	if !ok {
		termType = "linux"
	}

	err = c.session.RequestPty(termType, termHeight, termWidth, ssh.TerminalModes{})
	if err != nil {
		return err
	}

	c.updateTerminalSize()

	c.stdin, err = c.session.StdinPipe()
	if err != nil {
		return err
	}
	c.stdout, err = c.session.StdoutPipe()
	if err != nil {
		return err
	}
	c.stderr, err = c.session.StderrPipe()

	go io.Copy(os.Stderr, c.stderr)
	go io.Copy(os.Stdout, c.stdout)
	go func() {
		buf := make([]byte, 128)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				fmt.Println(err)
				return
			}
			if n > 0 {
				_, err = c.stdin.Write(buf[:n])
				if err != nil {
					fmt.Println(err)
					c.exitMsg = err.Error()
					return
				}
			}
		}
	}()

	err = c.session.Shell()
	if err != nil {
		return err
	}
	err = c.session.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (c *SshClient) Run(cmd string, w io.Writer) error {
	// close session
	defer c.closeSession()
	reader, ReaderErr := c.session.StdoutPipe()
	if ReaderErr != nil {
		return ReaderErr
	}
	scanner := bufio.NewScanner(reader)
	go func(output io.Writer) {
		for scanner.Scan() {
			if _, e := fmt.Fprintln(output, scanner.Text()); e != nil {
				continue
			}
		}
	}(w)

	if err := c.session.Run(cmd); err != nil {
		return err
	}
	return nil
}

func (c *SshClient) Login() error {
	return c.interactiveSession()
}

func (c *SshClient) PushFile(src string, dst string) error {
	var (
		Realsrc string
		Realdst string
	)
	sftpClient, err := sftp.NewClient(c.client)
	defer sftpClient.Close()
	//Get RealPath
	Realsrc = LocalRealPath(src)
	Realdst = RemoteRealpath(dst, sftpClient)

	// open file
	srcFile, err := os.Open(Realsrc)
	defer srcFile.Close()
	if err != nil {
		return err
	}
	dstFile, err := sftpClient.Create(Realdst)
	defer dstFile.Close()
	//bar
	SrcStat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	bar := pb.New64(SrcStat.Size()).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(Realsrc))
	bar.Start()
	r := bar.NewProxyReader(srcFile)
	defer bar.Finish()
	if _, err := io.Copy(dstFile, r); err != nil {
		return err
	}

	return nil
}

func (c *SshClient) GetFile(src string, dst string) error {
	var (
		Realsrc string
		Realdst string
	)
	// new SftpClient
	sftpClient, err := sftp.NewClient(c.client)
	defer sftpClient.Close()
	Realsrc = RemoteRealpath(src, sftpClient)
	Realdst = LocalRealPath(dst)
	if err != nil {
		return err
	}
	// open SrcFile
	srcFile, err := sftpClient.Open(Realsrc)
	defer srcFile.Close()
	if err != nil {
		return err
	}
	//bar
	SrcStat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	bar := pb.New64(SrcStat.Size()).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(Realsrc))
	bar.Start()
	// open DstFile
	dstFile, err := os.Create(Realdst)
	defer dstFile.Close()
	w := io.MultiWriter(bar, dstFile)
	defer bar.Finish()
	if _, err := srcFile.WriteTo(w); err != nil {
		return err
	}

	return nil
}

func (c *SshClient) PushDir(src string, dst string) error {
	var (
		Realsrc string
		Realdst string
	)
	sftpClient, err := sftp.NewClient(c.client)
	defer sftpClient.Close()
	if err != nil {
		return err
	}
	Realsrc = LocalRealPath(src)
	Realdst = RemoteRealpath(dst, sftpClient)

	root, dir := path.Split(Realsrc)
	if err := os.Chdir(root); err != nil {
		return err
	}
	size := TotalSize(Realsrc)
	bar := pb.New64(size).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(Realsrc))
	bar.Start()
	defer bar.Finish()
	var wg sync.WaitGroup
	WalkErr := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		DstPath := path.Join(Realdst, p)
		switch {
		case info.IsDir():
			if e := sftpClient.Mkdir(DstPath); e != nil {
				return e
			}
		default:

			wg.Add(1)
			go func(wgroup *sync.WaitGroup, b *pb.ProgressBar, Srcfile string, Dstfile string) {
				defer wgroup.Done()
				s, _ := os.Open(Srcfile)
				defer s.Close()
				d, _ := sftpClient.Create(Dstfile)
				defer d.Close()
				i, _ := io.Copy(d, s)
				b.Add64(i)
			}(&wg, bar, p, DstPath)
		}
		wg.Wait()
		return err
	})

	if WalkErr != nil {
		return err
	}
	return nil
}

func (c *SshClient) GetDir(src string, dst string) error {
	var (
		Realsrc string
		Realdst string
	)
	// new SftpClient
	sftpClient, err := sftp.NewClient(c.client)
	defer sftpClient.Close()
	if err != nil {
		return err
	}
	Realsrc = RemoteRealpath(src, sftpClient)
	Realdst = LocalRealPath(dst)
	walker := sftpClient.Walk(Realsrc)
	//获取远程目录的大小
	size := func(c *sftp.Client) int64 {
		var ret int64
		TotalWalk := c.Walk(Realsrc)
		for TotalWalk.Step() {
			stat := TotalWalk.Stat()
			if !stat.IsDir() {
				ret += stat.Size()
			}
		}
		return ret
	}(sftpClient)
	bar := pb.New64(size).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(path.Base(Realsrc))
	bar.Start()
	defer bar.Finish()
	//同步远程目录到本地
	var wg sync.WaitGroup
	base := path.Dir(Realsrc)
	wg.Add(1)
	go func(w *fs.Walker, c *sftp.Client, g *sync.WaitGroup, b *pb.ProgressBar) {
		for w.Step() {
			pdst := strings.TrimPrefix(w.Path(), base)
			p := path.Join(Realdst, pdst)
			stats := w.Stat()
			switch {
			case walker.Err() != nil:
				panic(walker.Err())
			case stats.IsDir():
				os.Mkdir(p, 0755)
			default:
				files, _ := c.Open(w.Path())
				defer files.Close()
				ds, errs := os.Create(p)
				if errs != nil {
					panic(errs)
				}
				defer ds.Close()
				//io.Copy(ds,file)
				i, e := io.Copy(ds, files)
				if e != nil {
					fmt.Println(e)
				}
				b.Add64(i)
			}
		}
		g.Done()
	}(walker, sftpClient, &wg, bar)
	wg.Wait()
	return nil
}

func (c *SshClient) Get(src, dst string) error {
	var (
		Realsrc string
		Realdst string
	)
	sftpCli, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer sftpCli.Close()
	Realsrc = RemoteRealpath(src, sftpCli)
	Realdst = LocalRealPath(dst)
	state, Serr := sftpCli.Stat(Realsrc)
	if Serr != nil {
		return Serr
	}
	if state.IsDir() {
		return c.GetDir(Realsrc, Realdst)
	} else {
		Dstat, _ := os.Stat(Realdst)
		if Dstat.IsDir() {
			return c.GetFile(Realsrc, filepath.Join(Realdst, filepath.Base(src)))
		} else {
			return c.GetFile(Realsrc, Realdst)
		}
	}
}

func (c *SshClient) Push(src, dst string) error {
	var (
		Realsrc string
		Realdst string
	)

	Realsrc = LocalRealPath(src)
	Sstate, Serr := os.Stat(Realsrc)
	if Serr != nil {
		panic(Serr)
	}
	if Sstate.IsDir() {
		return c.PushDir(Realsrc, dst)
	} else {
		sftpCli, err := sftp.NewClient(c.client)
		if err != nil {
			return err
		}
		defer sftpCli.Close()
		Realdst = RemoteRealpath(dst, sftpCli)
		Dstat, err := sftpCli.Stat(Realdst)
		if err != nil {
			panic(err)
		}
		if Dstat.IsDir() {
			return c.PushFile(Realsrc, filepath.Join(Realdst, filepath.Base(Realsrc)))
		} else {
			return c.PushFile(Realsrc, Realdst)
		}
	}
}

func (c *SshClient) TunnelStart(Local, Remote NetworkConfig) error {
	listener, err := net.Listen(Local.Network, Local.Address)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		c.forward(conn, Remote)
	}
}

func (c *SshClient) forward(localConn net.Conn, remote NetworkConfig) {
	remoteConn, err := c.client.Dial(remote.Network, remote.Address)
	if err != nil {
		return
	}

	copyConn := func(writer, reader net.Conn) {
		defer writer.Close()
		defer reader.Close()

		_, err := io.Copy(writer, reader)
		if err != nil {
			return
		}
	}
	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func (c *SshClient) Close() error {
	c.session.Close()
	return c.client.Close()
}

func (c *SshClient) Proxy(auth AuthConfig) (Client, error) {
	conn, connErr := c.client.Dial(auth.Network, auth.Address)
	if connErr != nil {
		return nil, connErr
	}
	proxyCfg := &ssh.ClientConfig{
		User:            auth.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(auth.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(auth.ConnectTimeout) * time.Second,
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, auth.Address, proxyCfg)
	if err != nil {
		return nil, err
	}
	client := ssh.NewClient(ncc, chans, reqs)
	session, sessionErr := client.NewSession()
	if sessionErr != nil {
		return nil, sessionErr
	}
	return &SshClient{client: client, session: session}, nil
}
