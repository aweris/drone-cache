package sftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Backend TODO
type Backend struct {
	cacheRoot string
	client    *sftp.Client
}

// New creates a new sFTP backend.
func New(l log.Logger, c Config) (*Backend, error) {
	sshClient, err := getSSHClient(c)
	if err != nil {
		return nil, err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to ssh with sftp protocol %w", err)
	}

	level.Debug(l).Log("msg", "sftp backend", "config", fmt.Sprintf("%#v", c))

	return &Backend{client: sftpClient, cacheRoot: c.CacheRoot}, nil
}

// Get writes downloaded content to the given writer.
func (b *Backend) Get(ctx context.Context, p string, w io.Writer) error {
	absPath, err := filepath.Abs(filepath.Clean(filepath.Join(b.cacheRoot, p)))
	if err != nil {
		return fmt.Errorf("absolute path %w", err)
	}

	errCh := make(chan error)
	go func() {
		defer close(errCh)

		rc, err := b.client.Open(absPath)
		if err != nil {
			errCh <- fmt.Errorf("get the object %w", err)
		}
		defer rc.Close()

		_, err = io.Copy(w, rc)
		if err != nil {
			errCh <- fmt.Errorf("copy the object %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Put uploads contents of the given reader.
func (b *Backend) Put(ctx context.Context, p string, r io.Reader) error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		path := filepath.Clean(filepath.Join(b.cacheRoot, p))

		dir := filepath.Dir(path)
		if err := b.client.MkdirAll(dir); err != nil {
			errCh <- fmt.Errorf("create directory <%s> %w", dir, err)
		}

		w, err := b.client.Create(path)
		if err != nil {
			errCh <- fmt.Errorf("create cache file <%s> %w", path, err)
		}
		defer w.Close()

		if _, err := io.Copy(w, r); err != nil {
			errCh <- fmt.Errorf("write contents of reader to a file %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Helpers

// TODO: parameterize only what it actually needs
func getSSHClient(c Config) (*ssh.Client, error) {
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

// TODO: parameterize only what it actually needs
func getAuthMethod(c Config) ([]ssh.AuthMethod, error) {
	if c.Auth.Method == SSHAuthMethodPassword {
		return []ssh.AuthMethod{
			ssh.Password(c.Auth.Password),
		}, nil
	} else if c.Auth.Method == SSHAuthMethodPublicKeyFile {
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
