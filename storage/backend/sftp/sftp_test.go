package sftp

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
)

const defaultSFTPHost = "127.0.0.1"
const defaultSFTPPort = "22"

var host = getEnv("TEST_SFTP_HOST", defaultSFTPHost)
var port = getEnv("TEST_SFTP_PORT", defaultSFTPPort)

func TestSFTPTruth(t *testing.T) {
	cli, err := New(
		log.NewNopLogger(),
		Config{
			CacheRoot: "/upload",
			Username:  "foo",
			Auth: SSHAuth{
				Password: "pass",
				Method:   SSHAuthMethodPassword,
			},
			Host: host,
			Port: port,
		},
	)
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
	var buf bytes.Buffer
	if cli.Get(context.TODO(), "test3.t", &buf); err != nil {
		t.Fatal(err)
	}

	b, _ := ioutil.ReadAll(&buf)
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
