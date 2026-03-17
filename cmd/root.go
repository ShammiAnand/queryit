package cmd

import (
	"fmt"
	"os"

	"github.com/shammianand/queryit/internal/config"
	"github.com/shammianand/queryit/internal/tui"
	"github.com/spf13/cobra"
)

var (
	profileFlag string
	configFlag  string
	version     = "dev"
	commit      = "unknown"
	buildDate   = "unknown"
)

func SetVersion(v, c, d string) {
	version = v
	commit = c
	buildDate = d
}

var rootCmd = &cobra.Command{
	Use:   "queryit",
	Short: "A keyboard-driven TUI for PostgreSQL",
	Long: `queryit is a full-screen terminal UI for executing SQL queries
against PostgreSQL databases. Supports direct and SSH-tunneled connections,
multiple simultaneous tabs, autocomplete, and query history.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if profileFlag != "" {
			if _, ok := cfg.Profiles[profileFlag]; !ok {
				fmt.Fprintf(os.Stderr, "error: profile %q not found\n", profileFlag)
				os.Exit(1)
			}
		}

		return tui.Run(cfg, profileFlag)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "profile to auto-connect on launch")
	rootCmd.PersistentFlags().StringVar(&configFlag, "config", "", "override config file path")
	rootCmd.Flags().BoolP("version", "v", false, "print version and exit")

	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("queryit %s (commit %s, built %s)\n", version, commit, buildDate)
	},
}
