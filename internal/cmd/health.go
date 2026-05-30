package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check API health",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := rt().client.HealthCheck(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "status: %s\n", status)
			return nil
		},
	}
}
