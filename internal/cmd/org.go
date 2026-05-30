package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/builderhub/build-cli/internal/output"
)

func orgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "org",
		Aliases: []string{"organization", "organizations"},
		Short:   "Manage organizations",
	}
	cmd.AddCommand(
		orgListCmd(),
		orgGetCmd(),
		orgCreateCmd(),
		orgUpdateCmd(),
		orgDeleteCmd(),
		orgMembersCmd(),
	)
	return cmd
}

func orgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organizations",
		RunE: func(cmd *cobra.Command, args []string) error {
			orgs, err := rt().client.ListOrganizations(cmd.Context())
			if err != nil {
				return err
			}
			return output.PrintOrganizations(cmd.OutOrStdout(), rt().outputFmt, orgs)
		},
	}
}

func orgGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Get an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := rt().client.GetOrganization(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.PrintOrganization(cmd.OutOrStdout(), rt().outputFmt, org)
		},
	}
}

func orgCreateCmd() *cobra.Command {
	var name, slug, plan string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || slug == "" {
				return fmt.Errorf("--name and --slug are required")
			}
			if plan == "" {
				plan = "starter"
			}
			org, err := rt().client.CreateOrganization(cmd.Context(), name, slug, plan)
			if err != nil {
				return err
			}
			return output.PrintOrganization(cmd.OutOrStdout(), rt().outputFmt, org)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Organization name")
	cmd.Flags().StringVar(&slug, "slug", "", "Organization slug")
	cmd.Flags().StringVar(&plan, "plan", "starter", "Plan: starter, pro, enterprise")
	return cmd
}

func orgUpdateCmd() *cobra.Command {
	var name, slug, plan string
	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := rt().client.UpdateOrganization(cmd.Context(), args[0], name, slug, plan)
			if err != nil {
				return err
			}
			return output.PrintOrganization(cmd.OutOrStdout(), rt().outputFmt, org)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Organization name")
	cmd.Flags().StringVar(&slug, "slug", "", "Organization slug")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan: starter, pro, enterprise")
	return cmd
}

func orgDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete an organization and all builders",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmDestructive(cmd, yes, fmt.Sprintf("Delete organization %q and all builders?", args[0])); err != nil {
				return err
			}
			if err := rt().client.DeleteOrganization(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Organization deleted")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func orgMembersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "Organization membership",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list [org-id]",
		Short: "List organization members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			members, err := rt().client.ListOrganizationMembers(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.PrintOrganizationMembers(cmd.OutOrStdout(), rt().outputFmt, members)
		},
	})
	return cmd
}
