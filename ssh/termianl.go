package ssh

import (
	"fmt"
	"github.com/kr/fs"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	terminal "golang.org/x/term"
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

type Option func(*ClientType)

type ClientType struct {
	session *ssh.Session
	client  *ssh.Client
	pb      bool
}

func NewClient(conf *AuthConfig, option ...Option) (Client, error) {
	clientCfg, cfgErr := authConfig(conf)
	if cfgErr != nil {
		return nil, cfgErr
	}

	cli, err := ssh.Dial(conf.Network, conf.Address, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("connect error: %w", err)
	}
	session, getSessionErr := cli.NewSession()
	if getSessionErr != nil {
		return nil, getSessionErr
	}
	tp := &ClientType{client: cli, session: session}
	for _, opt := range option {
		opt(tp)
	}
	return tp, nil
}

func authConfig(conf *AuthConfig) (*ssh.ClientConfig, error) {
	auth := make([]ssh.AuthMethod, 0)
	if conf.Password != "" {
		auth = append(auth, ssh.Password(conf.Password))
	} else {
		if conf.PrivateKey == "" {
			conf.PrivateKey = "~/.ssh/id_rsa"
		}
		var (
			content []byte
			readErr error
		)
		_, err := os.Stat(conf.PrivateKey)
		if os.IsExist(err) {
			content, readErr = os.ReadFile(conf.PrivateKey)
			if readErr != nil {
				return nil, fmt.Errorf("open private key %s error: %w", conf.PrivateKey, readErr)
			}
		} else {
			content = []byte(conf.PrivateKey)
		}
		signer, parseErr := ssh.ParsePrivateKey(content)
		if parseErr != nil {
			return nil, fmt.Errorf("parse private key %s error: %w", conf.PrivateKey, parseErr)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	return &ssh.ClientConfig{
		User:            conf.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(conf.ConnectTimeout) * time.Second,
	}, nil
}

func totalSize(paths string) int64 {
	var Ret int64
	stat, _ := os.Stat(paths)
	switch {
	case stat.IsDir():
		if err := filepath.Walk(paths, func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			} else {
				s, _ := os.Stat(p)
				Ret = Ret + s.Size()
				return nil
			}
		}); err != nil {
			return 0
		}
		return Ret
	default:
		return stat.Size()
	}
}

func localRealPath(ph string) string {
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

func remoteRealpath(ph string, c *sftp.Client) string {
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

func (c *ClientType) progressBar(title string, total int64) *pb.ProgressBar {
	bar := pb.New64(total)
	bar.SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.Prefix(title)
	return bar
}

func (c *ClientType) closeSession() error {
	return c.session.Close()
}

func (c *ClientType) interactiveSession() error {
	defer c.closeSession()

	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("terminal.MakeRaw error: %w", err)
	}
	defer terminal.Restore(fd, state)

	termWidth, termHeight, getSizeErr := terminal.GetSize(fd)
	if getSizeErr != nil {
		return fmt.Errorf("terminal.GetSize error: %w", getSizeErr)
	}

	termType, ok := os.LookupEnv("TERM")
	if !ok {
		termType = "linux"
	}

	if requestPtyErr := c.session.RequestPty(termType, termHeight, termWidth, ssh.TerminalModes{}); requestPtyErr != nil {
		return fmt.Errorf("c.session.RequestPty error: %w", requestPtyErr)
	}
	changeSizeErr := make(chan error)
	go func() {
		for changeErr := range changeSizeErr {
			fmt.Fprintf(os.Stderr, "updateTerminalSize err: %v", changeErr)
		}
	}()
	go c.updateTerminalSize(fd, termWidth, termHeight, changeSizeErr)
	defer close(changeSizeErr)

	//c.stdin, err = c.session.StdinPipe()
	//if err != nil {
	//	return err
	//}
	//c.stdout, err = c.session.StdoutPipe()
	//if err != nil {
	//	return err
	//}
	//c.stderr, err = c.session.StderrPipe()
	//if err != nil {
	//	return err
	//}
	c.session.Stdout = os.Stdout
	c.session.Stderr = os.Stderr
	c.session.Stdin = os.Stdin
	//go io.Copy(os.Stderr, c.session.Stderr)
	//go io.Copy(os.Stdout, c.stdout)
	//go func() {
	//	buf := make([]byte, 128)
	//	for {
	//		n, err := os.Stdin.Read(buf)
	//		if err != nil {
	//			fmt.Println(err)
	//			return
	//		}
	//		if n > 0 {
	//			_, err = c.stdin.Write(buf[:n])
	//			if err != nil {
	//				fmt.Println(err)
	//				c.exitMsg = err.Error()
	//				return
	//			}
	//		}
	//	}
	//}()

	if err = c.session.Shell(); err != nil {
		return err
	}
	return c.session.Wait()
}

func (c *ClientType) Login() error {
	return c.interactiveSession()
}

func (c *ClientType) Run(cmd string, stdout, stderr io.Writer) error {
	// close session
	defer c.closeSession()
	c.session.Stdout = stdout
	c.session.Stderr = stderr
	if err := c.session.Run(cmd); err != nil {
		return err
	}
	return nil
}

func (c *ClientType) PushFile(src string, dst string) error {
	sftpClient, sftpErr := sftp.NewClient(c.client)
	if sftpErr != nil {
		return sftpErr
	}
	defer sftpClient.Close()
	RealSrc := localRealPath(src)
	RealDst := remoteRealpath(dst, sftpClient)
	srcFile, openErr := os.Open(RealSrc)
	if openErr != nil {
		return openErr
	}
	defer srcFile.Close()

	dstFile, sftpCreateErr := sftpClient.Create(RealDst)
	if sftpCreateErr != nil {
		return sftpCreateErr
	}
	defer dstFile.Close()
	SrcStat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	var reader io.Reader = srcFile
	if c.pb {
		title := path.Base(RealSrc)
		total := SrcStat.Size()
		bar := c.progressBar(title, total)
		bar.Start()
		reader = bar.NewProxyReader(srcFile)
		defer bar.Finish()
	}

	_, err = io.Copy(dstFile, reader)
	return err
}

func (c *ClientType) GetFile(src string, dst string) error {
	sftpClient, sftpErr := sftp.NewClient(c.client)
	if sftpErr != nil {
		return sftpErr
	}
	defer sftpClient.Close()
	RealSrc := remoteRealpath(src, sftpClient)
	RealDst := localRealPath(dst)

	srcFile, sftpOpenErr := sftpClient.Open(RealSrc)
	if sftpOpenErr != nil {
		return sftpOpenErr
	}
	defer srcFile.Close()
	SrcStat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	dstFile, dstCreateErr := os.Create(RealDst)
	if dstCreateErr != nil {
		return dstCreateErr
	}
	defer dstFile.Close()

	var writer io.Writer = dstFile
	if c.pb {
		title := path.Base(RealSrc)
		total := SrcStat.Size()
		bar := c.progressBar(title, total)
		bar.Start()
		defer bar.Finish()
		writer = io.MultiWriter(bar, dstFile)
	}
	_, err = srcFile.WriteTo(writer)
	return err
}

func (c *ClientType) PushDir(src string, dst string) error {
	sftpClient, sftpErr := sftp.NewClient(c.client)
	if sftpErr != nil {
		return sftpErr
	}
	defer sftpClient.Close()
	RealSrc := localRealPath(src)
	RealDst := remoteRealpath(dst, sftpClient)

	root, dir := path.Split(RealSrc)
	if err := os.Chdir(root); err != nil {
		return err
	}
	var bar *pb.ProgressBar
	if c.pb {
		total := totalSize(RealSrc)
		title := path.Base(RealSrc)
		bar = c.progressBar(title, total)
		bar.Start()
		defer bar.Finish()
	}

	var wg sync.WaitGroup
	return filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		DstPath := path.Join(RealDst, p)
		switch {
		case info.IsDir():
			if e := sftpClient.Mkdir(DstPath); e != nil {
				return e
			}
		default:
			wg.Add(1)
			go func(w *sync.WaitGroup, b *pb.ProgressBar, SrcFile string, DstFile string) {
				defer w.Done()
				s, _ := os.Open(SrcFile)
				defer s.Close()
				d, _ := sftpClient.Create(DstFile)
				defer d.Close()
				i, _ := io.Copy(d, s)
				if b != nil {
					b.Add64(i)
				}
			}(&wg, bar, p, DstPath)
		}
		wg.Wait()
		return err
	})
}

func (c *ClientType) GetDir(src string, dst string) error {
	sftpClient, sftpErr := sftp.NewClient(c.client)
	if sftpErr != nil {
		return sftpErr
	}
	defer sftpClient.Close()

	RealSrc := remoteRealpath(src, sftpClient)
	RealDst := localRealPath(dst)
	walker := sftpClient.Walk(RealSrc)
	var bar *pb.ProgressBar
	if c.pb {
		//获取远程目录的大小
		size := func(c *sftp.Client) int64 {
			var ret int64
			TotalWalk := c.Walk(RealSrc)
			for TotalWalk.Step() {
				stat := TotalWalk.Stat()
				if !stat.IsDir() {
					ret += stat.Size()
				}
			}
			return ret
		}(sftpClient)
		title := path.Base(RealSrc)
		bar = c.progressBar(title, size)
		bar.Start()
		defer bar.Finish()
	}

	//同步远程目录到本地
	var wg sync.WaitGroup
	base := path.Dir(RealSrc)
	wg.Add(1)
	go func(w *fs.Walker, c *sftp.Client, g *sync.WaitGroup, b *pb.ProgressBar) {
		for w.Step() {
			prefix := strings.TrimPrefix(w.Path(), base)
			p := path.Join(RealDst, prefix)
			stats := w.Stat()
			switch {
			case walker.Err() != nil:
				panic(walker.Err())
			case stats.IsDir():
				os.Mkdir(p, 0755)
			default:
				files, _ := c.Open(w.Path())
				ds, errs := os.Create(p)
				if errs != nil {
					panic(errs)
				}
				//io.Copy(ds,file)
				i, e := io.Copy(ds, files)
				if e != nil {
					fmt.Println(e)
				}
				ds.Close()
				files.Close()
				if b != nil {
					b.Add64(i)
				}
			}
		}
		g.Done()
	}(walker, sftpClient, &wg, bar)
	wg.Wait()
	return nil
}

func (c *ClientType) Get(src, dst string) error {

	sftpCli, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer sftpCli.Close()
	RealSrc := remoteRealpath(src, sftpCli)
	RealDst := localRealPath(dst)
	state, statErr := sftpCli.Stat(RealSrc)
	if statErr != nil {
		return statErr
	}
	if state.IsDir() {
		return c.GetDir(RealSrc, RealDst)
	} else {
		dstState, _ := os.Stat(RealDst)
		if dstState.IsDir() {
			return c.GetFile(RealSrc, filepath.Join(RealDst, filepath.Base(src)))
		} else {
			return c.GetFile(RealSrc, RealDst)
		}
	}
}

func (c *ClientType) Push(src, dst string) error {

	RealSrc := localRealPath(src)
	SrcState, statErr := os.Stat(RealSrc)
	if statErr != nil {
		panic(statErr)
	}
	if SrcState.IsDir() {
		return c.PushDir(RealSrc, dst)
	} else {
		sftpCli, err := sftp.NewClient(c.client)
		if err != nil {
			return err
		}
		defer sftpCli.Close()
		RealDst := remoteRealpath(dst, sftpCli)
		dstErr, err := sftpCli.Stat(RealDst)
		if err != nil {
			panic(err)
		}
		if dstErr.IsDir() {
			return c.PushFile(RealSrc, filepath.Join(RealDst, filepath.Base(RealSrc)))
		} else {
			return c.PushFile(RealSrc, RealDst)
		}
	}
}

func (c *ClientType) TunnelStart(Local, Remote NetworkConfig) error {
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

func (c *ClientType) forward(localConn net.Conn, remote NetworkConfig) {
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

func (c *ClientType) Close() error {
	c.session.Close()
	return c.client.Close()
}

func (c *ClientType) Proxy(auth *AuthConfig) (Client, error) {
	conn, connErr := c.client.Dial(auth.Network, auth.Address)
	if connErr != nil {
		return nil, connErr
	}
	proxyCfg, cfgErr := authConfig(auth)
	if cfgErr != nil {
		return nil, cfgErr
	}
	ncc, cs, reqs, err := ssh.NewClientConn(conn, auth.Address, proxyCfg)
	if err != nil {
		return nil, err
	}
	client := ssh.NewClient(ncc, cs, reqs)
	session, sessionErr := client.NewSession()
	if sessionErr != nil {
		return nil, sessionErr
	}
	return &ClientType{client: client, session: session, pb: c.pb}, nil
}

func WithProgressBar(show bool) Option {
	return func(c *ClientType) {
		c.pb = show
	}
}
