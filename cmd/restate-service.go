package cmd

import (
	"github.com/curaious/uno/internal/api"
	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/telemetry"
	"github.com/spf13/cobra"
)

var restateWorkerCmd = &cobra.Command{
	Use:   "restate-worker",
	Short: "Start Restate Worker",
	Run: func(cmd *cobra.Command, args []string) {
		conf := config.ReadConfig()

		shutdownTelemetry := telemetry.NewProvider(conf.OTEL_EXPORTER_OTLP_ENDPOINT)
		defer shutdownTelemetry()

		s := api.New()
		s.StartRestateWorker()
	},
}

// Register the "server" command
func init() {
	rootCmd.AddCommand(restateWorkerCmd)
}
