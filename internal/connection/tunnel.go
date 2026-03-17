package connection

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

type Tunnel struct {
	listener  net.Listener
	sshClient *ssh.Client
	LocalPort int
}

func NewTunnel(bastionUser, bastionHost, pemPath, dbHost string, dbPort int) (*Tunnel, error) {
	pemPath = expandHome(pemPath)
	key, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, fmt.Errorf("read PEM file %q: %w", pemPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse PEM key: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User:            bastionUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
	}

	if !hasPort(bastionHost) {
		bastionHost = bastionHost + ":22"
	}
	sshClient, err := ssh.Dial("tcp", bastionHost, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %q: %w", bastionHost, err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("local listen: %w", err)
	}

	localPort := listener.Addr().(*net.TCPAddr).Port
	remoteAddr := fmt.Sprintf("%s:%d", dbHost, dbPort)

	go func() {
		for {
			local, err := listener.Accept()
			if err != nil {
				return
			}
			go forward(local, sshClient, remoteAddr)
		}
	}()

	return &Tunnel{
		listener:  listener,
		sshClient: sshClient,
		LocalPort: localPort,
	}, nil
}

func forward(local net.Conn, client *ssh.Client, remoteAddr string) {
	remote, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		local.Close()
		return
	}
	go pipe(local, remote)
	go pipe(remote, local)
}

func pipe(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	dst.Close()
	src.Close()
}

func (t *Tunnel) Close() {
	if t.listener != nil {
		t.listener.Close()
	}
	if t.sshClient != nil {
		t.sshClient.Close()
	}
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func hasPort(host string) bool {
	_, _, err := net.SplitHostPort(host)
	return err == nil
}
