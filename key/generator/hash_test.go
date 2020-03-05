package generator

import "testing"

func TestGenerateHash(t *testing.T) {
	// Implement me!
	t.Skip("skipping unimplemented test.")
}

func TestHash(t *testing.T) {
	t.Parallel()

	actual, err := hash("hash")
	if err != nil {
		t.Errorf("hash failed, error: %v\n", err)
	}

	expected := "0800fc577294c34e0b28ad2839435945"
	if actual != expected {
		t.Errorf("hash failed, got: %s, want: %s\n", actual, expected)
	}
}
