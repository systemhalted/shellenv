package cli

// version is stamped at link time via
// -ldflags "-X github.com/systemhalted/shellenv/internal/cli.version=v0.1.0";
// "dev" identifies unstamped builds (plain `go build`/`go run`).
var version = "dev"

func init() { rootCmd.Version = version }
