package cmd

import (
	"context"
	"log"

	"github.com/curaious/uno/internal/api"
	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/telemetry"
	"github.com/curaious/uno/service"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
	"github.com/spf13/cobra"
)

var agentServerCmd = &cobra.Command{
	Use:   "agent-server",
	Short: "Start Agent Server",
	Run: func(cmd *cobra.Command, args []string) {
		conf := config.ReadConfig()

		shutdownTelemetry := telemetry.NewProvider(conf.OTEL_EXPORTER_OTLP_ENDPOINT)
		defer shutdownTelemetry()

		go func() {
			if err := server.NewRestate().
				Bind(restate.Reflect(service.AgentBuilderWorkflow{})).
				Start(context.Background(), "0.0.0.0:9080"); err != nil {
				log.Fatal(err)
			}
		}()

		// Create the MCP Server
		s := api.New()
		s.Start()
	},
}

// Register the "server" command
func init() {
	rootCmd.AddCommand(agentServerCmd)
}
