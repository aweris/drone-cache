package sftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/meltwater/drone-cache/storage/backend"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sftpBackend struct {
	cacheRoot string
	client    *sftp.Client
}

// New TODO
func New(l log.Logger, cfgs backend.Configs) (backend.Backend, error) {
	level.Warn(l).Log("msg", "using sftp as backend")
	l = log.With(l, "backend", backend.SFTP)
	c := cfgs.SFTP

	sshClient, err := getSSHClient(c)
	if err != nil {
		return nil, err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to ssh with sftp protocol %w", err)
	}

	level.Debug(l).Log("msg", "sftp backend", "config", fmt.Sprintf("%#v", c))

	return &sftpBackend{client: sftpClient, cacheRoot: c.CacheRoot}, nil
}

// Get TODO
func (s sftpBackend) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	absPath, err := filepath.Abs(filepath.Clean(filepath.Join(s.cacheRoot, path)))
	if err != nil {
		return nil, fmt.Errorf("get the object %w", err)
	}

	return s.client.Open(absPath)
}

// Put TODO
func (s sftpBackend) Put(ctx context.Context, path string, src io.ReadSeeker) error {
	pathJoin := filepath.Join(s.cacheRoot, path)

	dir := filepath.Dir(pathJoin)
	if err := s.client.MkdirAll(dir); err != nil {
		return fmt.Errorf("create directory <%s> %w", dir, err)
	}

	dst, err := s.client.Create(pathJoin)
	if err != nil {
		return fmt.Errorf("create cache file <%s> %w", pathJoin, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("write read seeker as file %w", err)
	}

	return nil
}

// Helpers

func getSSHClient(c backend.SFTPConfig) (*ssh.Client, error) {
	authMethod, err := getAuthMethod(c)
	if err != nil {
		return nil, fmt.Errorf("unable to get ssh auth method %w", err)
	}

	/* #nosec */
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", c.Host, c.Port), &ssh.ClientConfig{
		User:            c.Username,
		Auth:            authMethod,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // #nosec just a workaround for now, will fix
	})
	if err != nil {
		return nil, fmt.Errorf("unable to connect to ssh %w", err)
	}

	return client, nil
}

func getAuthMethod(c backend.SFTPConfig) ([]ssh.AuthMethod, error) {
	if c.Auth.Method == backend.SSHAuthMethodPassword {
		return []ssh.AuthMethod{
			ssh.Password(c.Auth.Password),
		}, nil
	} else if c.Auth.Method == backend.SSHAuthMethodPublicKeyFile {
		pkAuthMethod, err := readPublicKeyFile(c.Auth.PublicKeyFile)
		return []ssh.AuthMethod{
			pkAuthMethod,
		}, err
	}

	return nil, errors.New("ssh method auth is not recognized, should be PASSWORD or PUBLIC_KEY_FILE")
}

func readPublicKeyFile(file string) (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read file <%s> %w", file, err)
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key %w", err)
	}

	return ssh.PublicKeys(key), nil
}
