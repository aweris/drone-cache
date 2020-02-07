package backend

type Configs struct {
	Debug bool

	S3         S3Config
	FileSystem FileSystemConfig
	SFTP       SFTPConfig
	Azure      AzureConfig
	GCS        GCSConfig
}

// Config configures behavior of Backend.
type Config interface {
	Apply(*Configs)
}

type configFunc func(*Configs)

func (f configFunc) Apply(c *Configs) {
	f(c)
}

// WithDebug sets debug flag.
func WithDebug(b bool) Config {
	return configFunc(func(c *Configs) {
		c.Debug = b
	})
}

// WithS3 sets debug flag.
func WithS3(cfg S3Config) Config {
	return configFunc(func(c *Configs) {
		c.S3 = cfg
	})
}

// WithFileSystem sets debug flag.
func WithFileSystem(cfg FileSystemConfig) Config {
	return configFunc(func(c *Configs) {
		c.FileSystem = cfg
	})
}

// WithAzure sets debug flag.
func WithAzure(cfg AzureConfig) Config {
	return configFunc(func(c *Configs) {
		c.Azure = cfg
	})
}

// WithSFTP sets debug flag.
func WithSFTP(cfg SFTPConfig) Config {
	return configFunc(func(c *Configs) {
		c.SFTP = cfg
	})
}

// WithGCS sets debug flag.
func WithGCS(cfg GCSConfig) Config {
	return configFunc(func(c *Configs) {
		c.GCS = cfg
	})
}
