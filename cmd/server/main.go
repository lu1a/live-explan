package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lu1a/live-explan/config"
	"github.com/lu1a/live-explan/internal/globals"
)

var rootCmd = &cobra.Command{
	Use:           "server",
	Short:         "Lewis Live Explanation",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Init()
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the server",

	Run: func(cmd *cobra.Command, args []string) {
		startServerDeps()
	},
}

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generates bash completion scripts",
	Long:  fmt.Sprintf("To load the completion: source <(%s completion)", os.Args[0]),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := rootCmd.GenBashCompletion(os.Stdout); err != nil {
			return fmt.Errorf("bash completion: %w", err)
		}
		return nil
	},
}

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",

	Run: func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(os.Stdout, "Version: %s\n", globals.Version)
		_, _ = fmt.Fprintf(os.Stdout, "Timestamp: %s\n", globals.VersionTime)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(VersionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %+v\n", err)
		os.Exit(1)
	}
}
