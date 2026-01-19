package cmd

import (
	"os"

	"github.com/curaious/uno/internal/api"
	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/telemetry"
	"github.com/spf13/cobra"
)

var temporalWorkerCmd = &cobra.Command{
	Use:   "temporal-worker",
	Short: "Start Temporal Worker",
	Run: func(cmd *cobra.Command, args []string) {
		conf := config.ReadConfig()

		os.Setenv("OTEL_SERVICE_NAME", "temporal-worker")

		shutdownTelemetry := telemetry.NewProvider(conf.OTEL_EXPORTER_OTLP_ENDPOINT)
		defer shutdownTelemetry()

		s := api.New()
		s.StartTemporalWorker()
	},
}

// Register the "server" command
func init() {
	rootCmd.AddCommand(temporalWorkerCmd)
}
