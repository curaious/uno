package cmd

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "gollm",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := godotenv.Load()
		if err != nil {
			log.Println("Error loading .env file, skipping")
		}
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err.Error())
	}
}
