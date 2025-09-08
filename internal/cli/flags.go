package cli

import "github.com/urfave/cli/v2"

// Common flags used across multiple commands
var (
	// Server configuration flags
	PortFlag = &cli.IntFlag{
		Name:    "port",
		Aliases: []string{"p"},
		EnvVars: []string{"PORT"},
		Usage:   "Port to run the server on",
		Value:   1995,
	}

	ServerTypeFlag = &cli.StringFlag{
		Name:    "server-type",
		Aliases: []string{"t"},
		EnvVars: []string{"SERVER_TYPE"},
		Usage:   "Server type: http,stdio",
		Value:   "stdio",
	}

	ServerHostnameFlag = &cli.StringFlag{
		Name:    "server-hostname",
		EnvVars: []string{"SERVER_HOSTNAME"},
		Usage:   "Server hostname",
		Value:   "localhost",
	}

	TmpDirFlag = &cli.StringFlag{
		Name:    "tmp-dir",
		EnvVars: []string{"TMP_DIR"},
		Usage:   "Temporary directory for provider binaries and state",
		Value:   "/tmp/spacelift-intent-mcp-executor",
	}

	// Spacelift connection flags
	SpaceliftURLFlag = &cli.StringFlag{
		Name:    "spacelift-url",
		EnvVars: []string{"SPACELIFT_URL"},
		Usage:   "Spacelift WebSocket URL to connect to",
		Value:   "localhost:9090",
	}

	SpaceliftAPIKeyIDFlag = &cli.StringFlag{
		Name:    "spacelift-api-key-id",
		EnvVars: []string{"SPACELIFT_API_KEY_ID"},
		Usage:   "Spacelift API key ID (required)",
	}

	SpaceliftAPIKeySecretFlag = &cli.StringFlag{
		Name:    "spacelift-api-key-secret",
		EnvVars: []string{"SPACELIFT_API_KEY_SECRET"},
		Usage:   "Spacelift API key secret (required)",
	}

	// Executor-specific flags
	SpaceliftConnectionSigningKeyFlag = &cli.StringFlag{
		Name:    "spacelift-connection-signing-key",
		EnvVars: []string{"SPACELIFT_CONNECTION_SIGNING_KEY"},
		Usage:   "Base64 encoded spacelift connection key. If not provided, the executor will not send any connection key to spacelift.",
		Value:   "",
	}

	ReconnectAttemptsFlag = &cli.StringFlag{
		Name:    "reconnect-attempts",
		EnvVars: []string{"RECONNECT_ATTEMPTS"},
		Usage:   "Number of reconnect attempts. If not provided, the executor will not attempt to reconnect to Spacelift.",
		Value:   "0",
	}

	ReconnectDelayFlag = &cli.StringFlag{
		Name:    "reconnect-delay",
		EnvVars: []string{"RECONNECT_DELAY"},
		Usage:   "Number of seconds to wait before attempting to reconnect to Spacelift.",
		Value:   "0",
	}

	ReconnectResetPeriodFlag = &cli.StringFlag{
		Name:    "reconnect-reset-period",
		EnvVars: []string{"RECONNECT_RESET_PERIOD"},
		Usage:   "Number of seconds to wait before resetting the reconnect attempt counter. Negative values mean never reset.",
		Value:   "-1",
	}

	// Observability flags
	ObservabilityVendorFlag = &cli.StringFlag{
		Name:    "observability-vendor",
		EnvVars: []string{"OBSERVABILITY_VENDOR"},
		Usage:   "Vendor to use for observability. Possible values are: disabled, aws, datadog, opentelemetry. If not provided, the executor will not send any observability data to the vendor.",
		Value:   "disabled",
	}

	ObservabilityServiceNameOverrideFlag = &cli.StringFlag{
		Name:     "observability-service-name-override",
		EnvVars:  []string{"OBSERVABILITY_SERVICE_NAME_OVERRIDE"},
		Usage:    "Name of service in observability, like traces",
		Required: false,
	}

	// Standalone-specific flags
	DBDirFlag = &cli.StringFlag{
		Name:    "db-dir",
		EnvVars: []string{"DB_DIR"},
		Usage:   "Directory containing DB files for persistent state",
		Value:   "./.state/",
	}
)

// FlagSet contains predefined flag collections for different command types
type FlagSet struct {
	flags []cli.Flag
}

// NewFlagSet creates a new flag set
func NewFlagSet(flags ...cli.Flag) *FlagSet {
	return &FlagSet{flags: flags}
}

// Add appends additional flags to the set
func (fs *FlagSet) Add(flags ...cli.Flag) *FlagSet {
	fs.flags = append(fs.flags, flags...)
	return fs
}

// Flags returns the flag slice for use with cli.App
func (fs *FlagSet) Flags() []cli.Flag {
	return fs.flags
}

// Common flag collections
func ExecutorFlags() []cli.Flag {
	return NewFlagSet(
		SpaceliftURLFlag,
		TmpDirFlag,
		SpaceliftConnectionSigningKeyFlag,
		ReconnectAttemptsFlag,
		ReconnectDelayFlag,
		ReconnectResetPeriodFlag,
		ObservabilityVendorFlag,
		ObservabilityServiceNameOverrideFlag,
	).Flags()
}

func MCPServerFlags() []cli.Flag {
	return NewFlagSet(
		PortFlag,
		ServerTypeFlag,
		SpaceliftURLFlag,
		SpaceliftAPIKeyIDFlag,
		SpaceliftAPIKeySecretFlag,
	).Flags()
}

func MCPServerFlagsWithPort(defaultPort int) []cli.Flag {
	portFlagWithDefault := &cli.IntFlag{
		Name:    PortFlag.Name,
		Aliases: PortFlag.Aliases,
		EnvVars: PortFlag.EnvVars,
		Usage:   PortFlag.Usage,
		Value:   defaultPort,
	}

	spaceliftURLFlagMCP := &cli.StringFlag{
		Name:     SpaceliftURLFlag.Name,
		EnvVars:  SpaceliftURLFlag.EnvVars,
		Usage:    "Spacelift intent-server WebSocket URL (required)",
		Required: true,
		Value:    "localhost:3008",
	}

	return NewFlagSet(
		portFlagWithDefault,
		ServerTypeFlag,
		ServerHostnameFlag,
		spaceliftURLFlagMCP,
		SpaceliftAPIKeyIDFlag,
		SpaceliftAPIKeySecretFlag,
	).Flags()
}

func StandaloneServerFlags() []cli.Flag {
	tmpDirFlagStandalone := &cli.StringFlag{
		Name:    TmpDirFlag.Name,
		EnvVars: TmpDirFlag.EnvVars,
		Usage:   "Temporary directory for provider binaries",
		Value:   "/tmp/spacelift-intent-mcp",
	}

	return NewFlagSet(
		PortFlag,
		ServerTypeFlag,
		tmpDirFlagStandalone,
		DBDirFlag,
	).Flags()
}
