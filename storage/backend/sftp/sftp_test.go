package sftp

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/meltwater/drone-cache/storage/backend"
)

const defaultSFTPHost = "127.0.0.1"
const defaultSFTPPort = "22"

var host = getEnv("TEST_SFTP_HOST", defaultSFTPHost)
var port = getEnv("TEST_SFTP_PORT", defaultSFTPPort)

func TestSFTPTruth(t *testing.T) {
	cli, err := New(log.NewNopLogger(),
		backend.Configs{
			SFTP: backend.SFTPConfig{
				CacheRoot: "/upload",
				Username:  "foo",
				Auth: backend.SSHAuth{
					Password: "pass",
					Method:   backend.SSHAuthMethodPassword,
				},
				Host: host,
				Port: port,
			},
			Debug: true,
		})
	if err != nil {
		t.Fatal(err)
	}

	content := "Hello world4"

	// PUT TEST
	file, _ := os.Create("test")
	_, _ = file.Write([]byte(content))
	_, _ = file.Seek(0, 0)
	err = cli.Put(context.TODO(), "test3.t", file)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	// GET TEST
	readCloser, err := cli.Get(context.TODO(), "test3.t")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := ioutil.ReadAll(readCloser)
	if !bytes.Equal(b, []byte(content)) {
		t.Fatal(string(b), "!=", content)
	}

	_ = os.Remove("test")
}

func getEnv(key, defaultVal string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return value
}
