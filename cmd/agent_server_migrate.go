package cmd

import (
	"fmt"
	"os"

	"github.com/praveen001/uno/internal/migrations"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run Migrations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(cmd.Help())
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display status of each migration",
	Run: func(cmd *cobra.Command, args []string) {
		migrator, err := migrations.NewMigrator()
		if err != nil {
			fmt.Println("Unable to initialize migrator", err)
			os.Exit(1)
		}

		if err := migrator.MigrationStatus(); err != nil {
			fmt.Println("Unable to fetch migration status", err)
			os.Exit(1)
		}
	},
}

var migrateCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new empty migration file",
	Run: func(cmd *cobra.Command, args []string) {
		migrator, err := migrations.NewMigrator()
		if err != nil {
			fmt.Println("Unable to initialize migrator", err)
			os.Exit(1)
		}

		name, err := cmd.Flags().GetString("name")
		if err != nil {
			fmt.Println("Unable to read flag `name`", err)
			os.Exit(1)
		}

		err = migrator.CreateMigration(name)
		if err != nil {
			fmt.Println("Unable to create new migration file", err)
			os.Exit(1)
		}

		os.Exit(0)
	},
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run up migrations",
	Long:  "Run all 'up' migrations by default.\nIf version is provided, it will run 'up' migrations to reach the version.\nIf step is provided, it will run `N` 'up' migrations.",
	Run: func(cmd *cobra.Command, args []string) {
		migrator, err := migrations.NewMigrator()
		if err != nil {
			fmt.Println("Unable to initialize migrator", err)
			os.Exit(1)
		}

		step, err := cmd.Flags().GetInt("step")
		if err != nil {
			fmt.Println("Unable to read flag `step`", err)
			os.Exit(1)
		}

		err = migrator.Up(step)
		if err != nil {
			fmt.Println("Unable to run `up` migrations", err)
			os.Exit(1)
		}

		os.Exit(0)
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Run down migrations",
	Long:  "Run all 'down' migrations by default.\nIf version is provided, it will run 'down' migrations to reach the version.\nIf step is provided, it will run `N` 'down' migrations.",
	Run: func(cmd *cobra.Command, args []string) {
		migrator, err := migrations.NewMigrator()
		if err != nil {
			fmt.Println("Unable to fetch migrator", err)
			os.Exit(1)
		}

		step, err := cmd.Flags().GetInt("step")
		if err != nil {
			fmt.Println("Unable to read flag `step`", err)
			os.Exit(1)
		}

		err = migrator.Down(step)
		if err != nil {
			fmt.Println("Unable to run `down` migrations", err)
			os.Exit(1)
		}

		os.Exit(0)
	},
}

// Register the "migrate" command
func init() {
	migrateCreateCmd.Flags().StringP("name", "n", "", "Name for the migration")
	migrateCmd.AddCommand(migrateCreateCmd)

	migrateUpCmd.Flags().IntP("step", "s", 0, "Number of migrations to execute")
	migrateCmd.AddCommand(migrateUpCmd)

	migrateDownCmd.Flags().IntP("step", "s", 0, "Number of migrations to execute")
	migrateCmd.AddCommand(migrateDownCmd)

	migrateCmd.AddCommand(migrateStatusCmd)

	rootCmd.AddCommand(migrateCmd)
}
