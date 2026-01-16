package cmd

import (
	"github.com/curaious/uno/internal/api"
	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/telemetry"
	"github.com/spf13/cobra"
)

var agentServerCmd = &cobra.Command{
	Use:   "agent-server",
	Short: "Start Agent Server",
	Run: func(cmd *cobra.Command, args []string) {
		conf := config.ReadConfig()

		shutdownTelemetry := telemetry.NewProvider(conf.OTEL_EXPORTER_OTLP_ENDPOINT)
		defer shutdownTelemetry()

		s := api.New()
		s.Start()
	},
}

// Register the "server" command
func init() {
	rootCmd.AddCommand(agentServerCmd)
}
