package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/builderhub/build-cli/internal/client"
	"github.com/builderhub/build-cli/internal/output"
)

func builderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "builder",
		Aliases: []string{"builders"},
		Short:   "Manage builders",
	}
	cmd.AddCommand(
		builderListCmd(),
		builderGetCmd(),
		builderCreateCmd(),
		builderUpdateCmd(),
		builderDeleteCmd(),
		builderWakeCmd(),
	)
	return cmd
}

type builderSpecFlags struct {
	mode        string
	replicas    int32
	idleTimeout int32
	templateRef string
	labels      []string
}

func (f builderSpecFlags) spec() (client.BuilderSpec, error) {
	spec := client.BuilderSpec{
		Mode:               f.mode,
		Replicas:           f.replicas,
		IdleTimeoutSeconds: f.idleTimeout,
		TemplateRef:        f.templateRef,
	}
	if len(f.labels) > 0 {
		spec.Labels = map[string]string{}
		for _, label := range f.labels {
			k, v, ok := strings.Cut(label, "=")
			if !ok || k == "" {
				return client.BuilderSpec{}, fmt.Errorf("invalid label %q (expected key=value)", label)
			}
			spec.Labels[k] = v
		}
	}
	return spec, nil
}

func addBuilderSpecFlags(cmd *cobra.Command, f *builderSpecFlags) {
	cmd.Flags().StringVar(&f.mode, "mode", "", "Builder mode: ephemeral, persistent, sleepy")
	cmd.Flags().Int32Var(&f.replicas, "replicas", 0, "Replica count")
	cmd.Flags().Int32Var(&f.idleTimeout, "idle-timeout", 0, "Idle timeout in seconds")
	cmd.Flags().StringVar(&f.templateRef, "template-ref", "", "Template reference")
	cmd.Flags().StringSliceVar(&f.labels, "label", nil, "Label key=value (repeatable)")
}

func builderListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List builders in an organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			builders, err := rt().client.ListBuilders(cmd.Context(), org)
			if err != nil {
				return err
			}
			return output.PrintBuilders(cmd.OutOrStdout(), rt().outputFmt, builders)
		},
	}
}

func builderGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [name]",
		Short: "Get a builder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			builder, err := rt().client.GetBuilder(cmd.Context(), org, args[0])
			if err != nil {
				return err
			}
			return output.PrintBuilder(cmd.OutOrStdout(), rt().outputFmt, builder)
		},
	}
}

func builderCreateCmd() *cobra.Command {
	var specFlags builderSpecFlags
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a builder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			if specFlags.mode == "" {
				return fmt.Errorf("--mode is required")
			}
			spec, err := specFlags.spec()
			if err != nil {
				return err
			}
			builder, err := rt().client.CreateBuilder(cmd.Context(), org, args[0], spec)
			if err != nil {
				return err
			}
			return output.PrintBuilder(cmd.OutOrStdout(), rt().outputFmt, builder)
		},
	}
	addBuilderSpecFlags(cmd, &specFlags)
	return cmd
}

func builderUpdateCmd() *cobra.Command {
	var specFlags builderSpecFlags
	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update a builder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			spec, err := specFlags.spec()
			if err != nil {
				return err
			}
			builder, err := rt().client.UpdateBuilder(cmd.Context(), org, args[0], spec)
			if err != nil {
				return err
			}
			return output.PrintBuilder(cmd.OutOrStdout(), rt().outputFmt, builder)
		},
	}
	addBuilderSpecFlags(cmd, &specFlags)
	return cmd
}

func builderDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a builder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			if err := confirmDestructive(cmd, yes, fmt.Sprintf("Delete builder %q?", args[0])); err != nil {
				return err
			}
			if err := rt().client.DeleteBuilder(cmd.Context(), org, args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Builder deleted")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func builderWakeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wake [name]",
		Short: "Wake a sleepy builder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			builder, err := rt().client.WakeBuilder(cmd.Context(), org, args[0])
			if err != nil {
				return err
			}
			return output.PrintBuilder(cmd.OutOrStdout(), rt().outputFmt, builder)
		},
	}
}
