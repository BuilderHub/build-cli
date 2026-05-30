package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/builderhub/build-cli/internal/config"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}
	cmd.AddCommand(configSetCmd(), configViewCmd())
	return cmd
}

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			if key == "profile" {
				rt().cfgFile.CurrentProfile = value
				if _, err := rt().cfgFile.Profile(value); err != nil {
					rt().cfgFile.SetProfile(value, config.Profile{APIURL: config.DefaultAPIURL})
				}
			} else {
				if err := rt().cfgFile.SetKey(rt().profileName, key, value); err != nil {
					return err
				}
			}
			if err := saveConfig(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s\n", key)
			return nil
		},
	}
}

func configViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "View current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ConfigPath()
			if err != nil {
				return err
			}
			profile, err := rt().cfgFile.Profile(rt().profileName)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n", path)
			fmt.Fprintf(cmd.OutOrStdout(), "Profile:     %s\n", rt().profileName)
			fmt.Fprintf(cmd.OutOrStdout(), "API URL:     %s\n", config.ResolveAPIURL(flagAPIURL, profile))
			fmt.Fprintf(cmd.OutOrStdout(), "Organization:%s\n", profile.Organization)
			if profile.AccessToken != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Auth:        JWT session")
			} else if profile.APIKey != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Auth:        API key")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Auth:        (none)")
			}
			return nil
		},
	}
}
