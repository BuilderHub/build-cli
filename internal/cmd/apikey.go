package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/builderhub/build-cli/internal/output"
)

func apiKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "api-key",
		Aliases: []string{"apikey"},
		Short:   "Manage user API keys",
	}
	cmd.AddCommand(apiKeyListCmd(), apiKeyCreateCmd(), apiKeyDeleteCmd())
	return cmd
}

func apiKeyListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireJWTSession()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := rt().client.ListUserAPIKeys(cmd.Context())
			if err != nil {
				return err
			}
			return output.PrintAPIKeys(cmd.OutOrStdout(), rt().outputFmt, keys)
		},
	}
}

func apiKeyCreateCmd() *cobra.Command {
	var name string
	var scopes []string
	var expiresInDays int
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new API key",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireJWTSession()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name = args[0]
			if len(scopes) == 0 {
				return fmt.Errorf("at least one --scope is required")
			}
			result, err := rt().client.CreateUserAPIKey(cmd.Context(), name, scopes, expiresInDays)
			if err != nil {
				return err
			}
			if err := output.PrintAPIKey(cmd.OutOrStdout(), rt().outputFmt, result.Key); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "\nSave this token now; it will not be shown again:\n%s\n", result.Token)
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&scopes, "scope", nil, "API key scope (repeatable)")
	cmd.Flags().IntVar(&expiresInDays, "expires-in-days", 0, "Days until expiry (-1 never, 0 server default)")
	return cmd
}

func apiKeyDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete [id]",
		Aliases: []string{"revoke"},
		Short:   "Revoke an API key",
		Args:    cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireJWTSession()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmDestructive(cmd, yes, fmt.Sprintf("Revoke API key %q?", args[0])); err != nil {
				return err
			}
			if err := rt().client.RevokeUserAPIKey(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "API key revoked")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
