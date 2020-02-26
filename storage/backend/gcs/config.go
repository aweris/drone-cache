package gcs

// Config is a structure to store Cloud Storage backend configuration
type Config struct {
	Bucket     string
	ACL        string
	Encryption string
	Endpoint   string
	APIKey     string
}
