package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/shammianand/queryit/internal/config"
	"github.com/shammianand/queryit/internal/tui"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage connection profiles",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if len(cfg.Profiles) == 0 {
			fmt.Println("No profiles configured.")
			return nil
		}
		names := make([]string, 0, len(cfg.Profiles))
		for n := range cfg.Profiles {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			p := cfg.Profiles[n]
			bastion := ""
			if p.Bastion != nil {
				bastion = fmt.Sprintf(" (via bastion %s@%s)", p.Bastion.User, p.Bastion.Host)
			}
			fmt.Printf("  %-20s %s:%d/%s%s\n", n, p.Host, p.Port, p.Database, bastion)
		}
		return nil
	},
}

var profileAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new connection profile (interactive)",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := tui.Prompt("Profile name")
		if name == "" {
			fmt.Fprintln(os.Stderr, "profile name is required")
			os.Exit(1)
		}

		host := tui.Prompt("Host")
		portStr := tui.Prompt("Port (default: 5432)")
		if portStr == "" {
			portStr = "5432"
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %w", err)
		}

		database := tui.Prompt("Database")
		user := tui.Prompt("User")
		password := tui.Prompt("Password (or $ENV_VAR)")
		sslmode := tui.Prompt("SSL mode (disable/require/prefer, default: prefer)")
		if sslmode == "" {
			sslmode = "prefer"
		}

		p := &config.Profile{
			Host:     host,
			Port:     port,
			Database: database,
			User:     user,
			Password: password,
			SSLMode:  sslmode,
		}

		bastionUser := tui.Prompt("Bastion SSH user (leave empty to skip)")
		if strings.TrimSpace(bastionUser) != "" {
			bastionHost := tui.Prompt("Bastion host")
			bastionPEM := tui.Prompt("Path to PEM file")
			p.Bastion = &config.BastionConfig{
				User: bastionUser,
				Host: bastionHost,
				PEM:  bastionPEM,
			}
		}

		if err := config.AddProfile(name, p); err != nil {
			return fmt.Errorf("save profile: %w", err)
		}
		fmt.Printf("Profile %q saved.\n", name)
		return nil
	},
}

var profileRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a connection profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.RemoveProfile(args[0]); err != nil {
			return err
		}
		fmt.Printf("Profile %q removed.\n", args[0])
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileRemoveCmd)
}
