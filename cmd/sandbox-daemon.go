package cmd

import (
	"os"

	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/telemetry"
	"github.com/curaious/uno/pkg/sandbox/daemon"
	"github.com/spf13/cobra"
)

var sandboxDaemonCmd = &cobra.Command{
	Use:   "sandbox-daemon",
	Short: "Start Sandbox Daemon",
	Run: func(cmd *cobra.Command, args []string) {
		conf := config.ReadConfig()

		os.Setenv("OTEL_SERVICE_NAME", "sandbox-daemon")

		shutdownTelemetry := telemetry.NewProvider(conf.OTEL_EXPORTER_OTLP_ENDPOINT)
		defer shutdownTelemetry()

		daemon.NewSandboxDaemon()
	},
}

// Register the "server" command
func init() {
	rootCmd.AddCommand(sandboxDaemonCmd)
}
