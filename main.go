package main

import (
	"errors"
	stdlog "log"
	"os"

	"github.com/meltwater/drone-cache/archive"
	"github.com/meltwater/drone-cache/internal"
	"github.com/meltwater/drone-cache/internal/metadata"
	"github.com/meltwater/drone-cache/internal/plugin"
	"github.com/meltwater/drone-cache/storage"
	"github.com/meltwater/drone-cache/storage/backend"
	"github.com/meltwater/drone-cache/storage/backend/azure"
	"github.com/meltwater/drone-cache/storage/backend/filesystem"
	"github.com/meltwater/drone-cache/storage/backend/gcs"
	"github.com/meltwater/drone-cache/storage/backend/s3"
	"github.com/meltwater/drone-cache/storage/backend/sftp"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/urfave/cli/v2"
)

var version = "0.0.0"

//nolint:funlen
func main() {
	app := cli.NewApp()
	app.Name = "Drone cache plugin"
	app.Usage = "Drone cache plugin"
	app.Action = run
	app.Version = version
	app.Flags = []cli.Flag{
		// Logger args

		&cli.StringFlag{
			Name:    "log.level, ll",
			Usage:   "log filtering level. ('error', 'warn', 'info', 'debug')",
			Value:   internal.LogLevelInfo,
			EnvVars: []string{"PLUGIN_LOG_LEVEL", "LOG_LEVEL"},
		},
		&cli.StringFlag{
			Name:    "log.format, lf",
			Usage:   "log format to use. ('logfmt', 'json')",
			Value:   internal.LogFormatLogfmt,
			EnvVars: []string{"PLUGIN_LOG_FORMAT", "LOG_FORMAT"},
		},

		// Repo args

		&cli.StringFlag{
			Name:    "repo.fullname, rf",
			Usage:   "repository full name",
			EnvVars: []string{"DRONE_REPO"},
		},
		&cli.StringFlag{
			Name:    "repo.namespace, rns",
			Usage:   "repository namespace",
			EnvVars: []string{"DRONE_REPO_NAMESPACE"},
		},
		&cli.StringFlag{
			Name:    "repo.owner, ro",
			Usage:   "repository owner (for Drone version < 1.0)",
			EnvVars: []string{"DRONE_REPO_OWNER"},
		},
		&cli.StringFlag{
			Name:    "repo.name, rn",
			Usage:   "repository name",
			EnvVars: []string{"DRONE_REPO_NAME"},
		},
		&cli.StringFlag{
			Name:    "repo.link, rl",
			Usage:   "repository link",
			EnvVars: []string{"DRONE_REPO_LINK"},
		},
		&cli.StringFlag{
			Name:    "repo.avatar, ra",
			Usage:   "repository avatar",
			EnvVars: []string{"DRONE_REPO_AVATAR"},
		},
		&cli.StringFlag{
			Name:    "repo.branch, rb",
			Usage:   "repository default branch",
			EnvVars: []string{"DRONE_REPO_BRANCH"},
		},
		&cli.BoolFlag{
			Name:    "repo.private, rp",
			Usage:   "repository is private",
			EnvVars: []string{"DRONE_REPO_PRIVATE"},
		},
		&cli.BoolFlag{
			Name:    "repo.trusted, rt",
			Usage:   "repository is trusted",
			EnvVars: []string{"DRONE_REPO_TRUSTED"},
		},

		// Commit args

		&cli.StringFlag{
			Name:    "remote.url, remu",
			Usage:   "git remote url",
			EnvVars: []string{"DRONE_REMOTE_URL"},
		},
		&cli.StringFlag{
			Name:    "commit.sha, cs",
			Usage:   "git commit sha",
			EnvVars: []string{"DRONE_COMMIT_SHA"},
		},
		&cli.StringFlag{
			Name:    "commit.ref, cr",
			Value:   "refs/heads/master",
			Usage:   "git commit ref",
			EnvVars: []string{"DRONE_COMMIT_REF"},
		},
		&cli.StringFlag{
			Name:    "commit.branch, cb",
			Value:   "master",
			Usage:   "git commit branch",
			EnvVars: []string{"DRONE_COMMIT_BRANCH"},
		},
		&cli.StringFlag{
			Name:    "commit.message, cm",
			Usage:   "git commit message",
			EnvVars: []string{"DRONE_COMMIT_MESSAGE"},
		},
		&cli.StringFlag{
			Name:    "commit.link, cl",
			Usage:   "git commit link",
			EnvVars: []string{"DRONE_COMMIT_LINK"},
		},
		&cli.StringFlag{
			Name:    "commit.author.name, an",
			Usage:   "git author name",
			EnvVars: []string{"DRONE_COMMIT_AUTHOR"},
		},
		&cli.StringFlag{
			Name:    "commit.author.email, ae",
			Usage:   "git author email",
			EnvVars: []string{"DRONE_COMMIT_AUTHOR_EMAIL"},
		},
		&cli.StringFlag{
			Name:    "commit.author.avatar, aa",
			Usage:   "git author avatar",
			EnvVars: []string{"DRONE_COMMIT_AUTHOR_AVATAR"},
		},

		// Build args

		&cli.StringFlag{
			Name:    "build.event, be",
			Value:   "push",
			Usage:   "build event",
			EnvVars: []string{"DRONE_BUILD_EVENT"},
		},
		&cli.IntFlag{
			Name:    "build.number, bn",
			Usage:   "build number",
			EnvVars: []string{"DRONE_BUILD_NUMBER"},
		},
		&cli.IntFlag{
			Name:    "build.created, bc",
			Usage:   "build created",
			EnvVars: []string{"DRONE_BUILD_CREATED"},
		},
		&cli.IntFlag{
			Name:    "build.started, bs",
			Usage:   "build started",
			EnvVars: []string{"DRONE_BUILD_STARTED"},
		},
		&cli.IntFlag{
			Name:    "build.finished, bf",
			Usage:   "build finished",
			EnvVars: []string{"DRONE_BUILD_FINISHED"},
		},
		&cli.StringFlag{
			Name:    "build.status, bstat",
			Usage:   "build status",
			Value:   "success",
			EnvVars: []string{"DRONE_BUILD_STATUS"},
		},
		&cli.StringFlag{
			Name:    "build.link, bl",
			Usage:   "build link",
			EnvVars: []string{"DRONE_BUILD_LINK"},
		},
		&cli.StringFlag{
			Name:    "build.deploy, db",
			Usage:   "build deployment target",
			EnvVars: []string{"DRONE_DEPLOY_TO"},
		},
		&cli.BoolFlag{
			Name:    "yaml.verified, yv",
			Usage:   "build yaml is verified",
			EnvVars: []string{"DRONE_YAML_VERIFIED"},
		},
		&cli.BoolFlag{
			Name:    "yaml.signed, ys",
			Usage:   "build yaml is signed",
			EnvVars: []string{"DRONE_YAML_SIGNED"},
		},

		// Prev build args

		&cli.IntFlag{
			Name:    "prev.build.number, pbn",
			Usage:   "previous build number",
			EnvVars: []string{"DRONE_PREV_BUILD_NUMBER"},
		},
		&cli.StringFlag{
			Name:    "prev.build.status, pbst",
			Usage:   "previous build status",
			EnvVars: []string{"DRONE_PREV_BUILD_STATUS"},
		},
		&cli.StringFlag{
			Name:    "prev.commit.sha, pcs",
			Usage:   "previous build sha",
			EnvVars: []string{"DRONE_PREV_COMMIT_SHA"},
		},

		// Config args

		&cli.StringFlag{
			Name:    "backend, b",
			Usage:   "cache backend to use in plugin (s3, filesystem, sftp, azure, gcs)",
			Value:   backend.S3,
			EnvVars: []string{"PLUGIN_BACKEND"},
		},
		&cli.StringSliceFlag{
			Name:    "mount, m",
			Usage:   "cache directories, an array of folders to cache",
			EnvVars: []string{"PLUGIN_MOUNT"},
		},
		&cli.BoolFlag{
			Name:    "rebuild, reb",
			Usage:   "rebuild the cache directories",
			EnvVars: []string{"PLUGIN_REBUILD"},
		},
		&cli.BoolFlag{
			Name:    "restore, res",
			Usage:   "restore the cache directories",
			EnvVars: []string{"PLUGIN_RESTORE"},
		},
		// RESTORE
		&cli.StringFlag{
			Name:    "cache-key, chk",
			Usage:   "cache key to use for the cache directories",
			EnvVars: []string{"PLUGIN_CACHE_KEY"},
		},
		// CACHE-KEYS
		// REBUILD-KEYS
		// RESTORE-KEYS
		&cli.StringFlag{
			Name:    "archive-format, arcfmt",
			Usage:   "archive format to use to store the cache directories (tar, gzip)",
			Value:   archive.DefaultArchiveFormat,
			EnvVars: []string{"PLUGIN_ARCHIVE_FORMAT"},
		},
		&cli.IntFlag{
			Name: "compression-level, cpl",
			Usage: `compression level to use for gzip compression when archive-format specified as gzip
			(check https://godoc.org/compress/flate#pkg-constants for available options)`,
			Value:   archive.DefaultCompressionLevel,
			EnvVars: []string{"PLUGIN_COMPRESSION_LEVEL"},
		},
		&cli.BoolFlag{
			Name:    "skip-symlinks, ss",
			Usage:   "skip symbolic links in archive",
			EnvVars: []string{"PLUGIN_SKIP_SYMLINKS", "SKIP_SYMLINKS"},
		},
		&cli.DurationFlag{
			Name:    "storage-operation-timeout, stopt",
			Usage:   "timeout value to use for each storage operations",
			Value:   storage.DefaultOperationTimeout,
			EnvVars: []string{"STORAGE_OPERATION_TIMEOUT"},
		},
		&cli.BoolFlag{
			Name:    "debug, d",
			Usage:   "debug",
			EnvVars: []string{"PLUGIN_DEBUG, DEBUG"},
		},
		&cli.BoolFlag{
			Name:    "exit-code, ex",
			Usage:   "always exit with exit code, disable silent fails for known errors",
			Hidden:  true,
			EnvVars: []string{"PLUGIN_EXIT_CODE", "EXIT_CODE"},
		},

		// Volume specific Config args

		&cli.StringFlag{
			Name:    "filesystem-cache-root, fcr",
			Usage:   "local filesystem root directory for the filesystem cache",
			Value:   "/tmp/cache",
			EnvVars: []string{"PLUGIN_FILESYSTEM_CACHE_ROOT", "FILESYSTEM_CACHE_ROOT"},
		},

		// S3 specific Config args

		&cli.StringFlag{
			Name:    "endpoint, e",
			Usage:   "endpoint for the s3/cloud storage connection",
			EnvVars: []string{"PLUGIN_ENDPOINT", "S3_ENDPOINT", "CLOUD_STORAGE_ENDPOINT"},
		},
		&cli.StringFlag{
			Name:    "access-key, akey",
			Usage:   "AWS access key",
			EnvVars: []string{"PLUGIN_ACCESS_KEY", "AWS_ACCESS_KEY_ID", "CACHE_AWS_ACCESS_KEY_ID"},
		},
		&cli.StringFlag{
			Name:    "secret-key, skey",
			Usage:   "AWS/GCP secret key",
			EnvVars: []string{"PLUGIN_SECRET_KEY", "AWS_SECRET_ACCESS_KEY", "CACHE_AWS_SECRET_ACCESS_KEY", "GCP_API_KEY"},
		},
		&cli.StringFlag{
			Name:    "bucket, bckt",
			Usage:   "AWS bucket name",
			EnvVars: []string{"PLUGIN_BUCKET", "S3_BUCKET", "CLOUD_STORAGE_BUCKET"},
		},
		&cli.StringFlag{
			Name:    "region, reg",
			Usage:   "AWS bucket region. (us-east-1, eu-west-1, ...)",
			EnvVars: []string{"PLUGIN_REGION", "S3_REGION"},
		},
		&cli.BoolFlag{
			Name:    "path-style, ps",
			Usage:   "use path style for bucket paths. (true for minio, false for aws)",
			EnvVars: []string{"PLUGIN_PATH_STYLE"},
		},
		&cli.StringFlag{
			Name:    "acl",
			Usage:   "upload files with acl (private, public-read, ...)",
			Value:   "private",
			EnvVars: []string{"PLUGIN_ACL"},
		},
		&cli.StringFlag{
			Name:    "encryption, enc",
			Usage:   "server-side encryption algorithm, defaults to none. (AES256, aws:kms)",
			EnvVars: []string{"PLUGIN_ENCRYPTION"},
		},

		// Azure specific Config flags

		&cli.StringFlag{
			Name:    "azure-account-name",
			Usage:   "Azure Blob Storage Account Name",
			EnvVars: []string{"PLUGIN_ACCOUNT_NAME", "AZURE_ACCOUNT_NAME"},
		},
		&cli.StringFlag{
			Name:    "azure-account-key",
			Usage:   "Azure Blob Storage Account Key",
			EnvVars: []string{"PLUGIN_ACCOUNT_KEY", "AZURE_ACCOUNT_KEY"},
		},
		&cli.StringFlag{
			Name:    "azure-blob-container-name",
			Usage:   "Azure Blob Storage container name",
			EnvVars: []string{"PLUGIN_CONTAINER", "AZURE_CONTAINER_NAME"},
		},
		&cli.StringFlag{
			Name:    "azure-blob-storage-url",
			Usage:   "Azure Blob Storage URL",
			Value:   "blob.core.windows.net",
			EnvVars: []string{"AZURE_BLOB_STORAGE_URL"},
		},
		&cli.IntFlag{
			Name:    "azure-blob-max-retry-requets",
			Usage:   "Azure Blob Storage Max Retry Requests",
			EnvVars: []string{"AZURE_BLOB_MAX_RETRY_REQUESTS"},
			Value:   azure.DefaultBlobMaxRetryRequests,
		},

		// SFTP specific Config flags

		&cli.StringFlag{
			Name:    "sftp-cache-root",
			Usage:   "sftp root directory",
			EnvVars: []string{"SFTP_CACHE_ROOT"},
		},
		&cli.StringFlag{
			Name:    "sftp-username",
			Usage:   "sftp username",
			EnvVars: []string{"SFTP_USERNAME"},
		},
		&cli.StringFlag{
			Name:    "sftp-password",
			Usage:   "sftp password",
			EnvVars: []string{"SFTP_PASSWORD"},
		},
		&cli.StringFlag{
			Name:    "ftp-public-key-file",
			Usage:   "sftp public key file path",
			EnvVars: []string{"SFTP_PUBLIC_KEY_FILE"},
		},
		&cli.StringFlag{
			Name:    "sftp-auth-method",
			Usage:   "sftp auth method, defaults to none. (PASSWORD, PUBLIC_KEY_FILE)",
			EnvVars: []string{"SFTP_AUTH_METHOD"},
		},
		&cli.StringFlag{
			Name:    "sftp-host",
			Usage:   "sftp host",
			EnvVars: []string{"SFTP_HOST"},
		},
		&cli.StringFlag{
			Name:    "sftp-port",
			Usage:   "sftp port",
			EnvVars: []string{"SFTP_PORT"},
		},
	}

	if err := app.Run(os.Args); err != nil {
		stdlog.Fatalf("%#v", err)
	}
}

//nolint:funlen
func run(c *cli.Context) error {
	var logLevel = c.String("log.level")
	if c.Bool("debug") {
		logLevel = internal.LogLevelDebug
	}

	logger := internal.NewLogger(logLevel, c.String("log.format"), "drone-cache")

	plg := plugin.New(log.With(logger, "component", "plugin"))
	plg.Metadata = metadata.Metadata{
		Repo: metadata.Repo{
			Namespace: c.String("repo.namespace"),
			Owner:     c.String("repo.owner"),
			Name:      c.String("repo.name"),
			Link:      c.String("repo.link"),
			Avatar:    c.String("repo.avatar"),
			Branch:    c.String("repo.branch"),
			Private:   c.Bool("repo.private"),
			Trusted:   c.Bool("repo.trusted"),
		},
		Build: metadata.Build{
			Number:   c.Int("build.number"),
			Event:    c.String("build.event"),
			Status:   c.String("build.status"),
			Deploy:   c.String("build.deploy"),
			Created:  int64(c.Int("build.created")),
			Started:  int64(c.Int("build.started")),
			Finished: int64(c.Int("build.finished")),
			Link:     c.String("build.link"),
		},
		Commit: metadata.Commit{
			Remote:  c.String("remote.url"),
			Sha:     c.String("commit.sha"),
			Ref:     c.String("commit.sha"),
			Link:    c.String("commit.link"),
			Branch:  c.String("commit.branch"),
			Message: c.String("commit.message"),
			Author: metadata.Author{
				Name:   c.String("commit.author.name"),
				Email:  c.String("commit.author.email"),
				Avatar: c.String("commit.author.avatar"),
			},
		},
	}

	plg.Config = plugin.Config{
		ArchiveFormat:           c.String("archive-format"),
		Backend:                 c.String("backend"),
		CacheKeyTemplate:        c.String("cache-key"),
		CompressionLevel:        c.Int("compression-level"),
		Debug:                   c.Bool("debug"),
		Mount:                   c.StringSlice("mount"),
		Rebuild:                 c.Bool("rebuild"),
		Restore:                 c.Bool("restore"),
		StorageOperationTimeout: c.Duration("storage-operation-timeout"),

		FileSystem: filesystem.Config{
			CacheRoot: c.String("filesystem-cache-root"),
		},

		S3: s3.Config{
			ACL:        c.String("acl"),
			Bucket:     c.String("bucket"),
			Encryption: c.String("encryption"),
			Endpoint:   c.String("endpoint"),
			Key:        c.String("access-key"),
			PathStyle:  c.Bool("path-style"),
			Region:     c.String("region"),
			Secret:     c.String("secret-key"),
		},

		Azure: azure.Config{
			AccountName:    c.String("azure-account-name"),
			AccountKey:     c.String("azure-account-key"),
			ContainerName:  c.String("azure-container-name"),
			BlobStorageURL: c.String("azure-blob-storage-url"),
			Azurite:        false,
		},

		SFTP: sftp.Config{
			CacheRoot: c.String("sftp-cache-root"),
			Username:  c.String("sftp-username"),
			Host:      c.String("sftp-host"),
			Port:      c.String("sftp-port"),
			Auth: sftp.SSHAuth{
				Password:      c.String("sftp-password"),
				PublicKeyFile: c.String("sftp-public-key-file"),
				Method:        sftp.SSHAuthMethod(c.String("sftp-auth-method")),
			},
		},

		GCS: gcs.Config{
			Bucket:     c.String("bucket"),
			Encryption: c.String("encryption"),
			Endpoint:   c.String("endpoint"),
			APIKey:     c.String("secret-key"),
		},

		SkipSymlinks: c.Bool("skip-symlinks"),
	}

	err := plg.Exec()
	if err == nil {
		return nil
	}

	if c.Bool("exit-code") {
		// If it is exit-code enabled, always exit with error.
		level.Warn(logger).Log("msg", "silent fails disabled, exiting with status code on error")

		return err
	}

	var e plugin.Error
	if errors.As(err, &e) {
		// If it is an expected error log it, handle it gracefully,
		level.Error(logger).Log("err", err)

		return nil
	}

	return err
}
