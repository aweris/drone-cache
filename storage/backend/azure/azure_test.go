package azure

import (
	"bytes"
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
)

const defaultBlobStorageURL = "127.0.0.1:10000"

var blobURL = getEnv("TEST_AZURITE_URL", defaultBlobStorageURL)

func TestAzureTruth(t *testing.T) {

	b, err := New(
		log.NewNopLogger(),
		Config{
			AccountName:    "devstoreaccount1",
			AccountKey:     "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==",
			ContainerName:  "testcontainer",
			BlobStorageURL: blobURL,
			Azurite:        true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	token := make([]byte, 32)
	rand.Read(token)
	testData := bytes.NewReader(token)

	// PUT TEST
	err = b.Put(context.TODO(), "test_key", testData)
	if err != nil {
		t.Fatal(err)
	}

	// GET TEST
	var buf bytes.Buffer
	if err := b.Get(context.TODO(), "test_key", &buf); err != nil {
		t.Fatal(err)
	}

	// Check the validity of returned bytes
	readData, err := ioutil.ReadAll(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(readData, token) {
		t.Error(string(readData), "!=", token)
	}
}

func getEnv(key, defaultVal string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return value
}
