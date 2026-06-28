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
		builderCredentialsCmd(),
		builderConnectCmd(),
	)
	return cmd
}

type builderSpecFlags struct {
	mode        string
	replicas    int32
	idleTimeout int32
	templateRef string
	labels      []string
	expose      bool
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
	cmd.Flags().StringVar(&f.mode, "mode", "", "Builder mode (persistent or sleepy)")
	cmd.Flags().Int32Var(&f.replicas, "replicas", 0, "Replica count")
	cmd.Flags().Int32Var(&f.idleTimeout, "idle-timeout", 0, "Idle timeout in seconds")
	cmd.Flags().StringVar(&f.templateRef, "template-ref", "", "Template reference (required for create; use 'template list' to see options)")
	cmd.Flags().StringSliceVar(&f.labels, "label", nil, "Label key=value (repeatable)")
	cmd.Flags().BoolVar(&f.expose, "expose", false, "Expose builder to the internet via ingress")
}

func boolPtr(v bool) *bool {
	return &v
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
	var connect, setDefault bool
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a builder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			if setDefault && !connect {
				return fmt.Errorf("--default requires --connect")
			}
			if specFlags.mode == "" {
				return fmt.Errorf("--mode is required")
			}
			if specFlags.mode != "persistent" && specFlags.mode != "sleepy" {
				return fmt.Errorf("--mode must be one of: persistent, sleepy")
			}
			if specFlags.templateRef == "" {
				return fmt.Errorf("--template-ref is required (create a template first with 'template create')")
			}
			spec, err := specFlags.spec()
			if err != nil {
				return err
			}
			expose := specFlags.expose || connect
			if expose {
				spec.Expose = boolPtr(true)
			}
			builder, err := rt().client.CreateBuilder(cmd.Context(), org, args[0], spec)
			if err != nil {
				return err
			}
			if err := output.PrintBuilder(cmd.OutOrStdout(), rt().outputFmt, builder); err != nil {
				return err
			}
			if connect {
				if err := requireJWTSession(); err != nil {
					return err
				}
				return connectBuilder(cmd.Context(), org, args[0], connectBuilderOpts{
					wait:       true,
					setDefault: setDefault,
				})
			}
			return nil
		},
	}
	addBuilderSpecFlags(cmd, &specFlags)
	cmd.Flags().BoolVar(&connect, "connect", false, "After create, mint mTLS credentials and configure local docker buildx")
	cmd.Flags().BoolVar(&setDefault, "default", false, "With --connect, set the buildx builder as default")
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
			if specFlags.mode != "" && specFlags.mode != "persistent" && specFlags.mode != "sleepy" {
				return fmt.Errorf("--mode must be one of: persistent, sleepy")
			}
			spec, err := specFlags.spec()
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("expose") {
				spec.Expose = boolPtr(specFlags.expose)
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

func builderCredentialsCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "credentials [name]",
		Short: "Generate new mTLS client credentials for an exposed builder",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireJWTSession()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			creds, err := rt().client.GenerateBuilderCredentials(cmd.Context(), org, args[0])
			if err != nil {
				return err
			}
			if rt().outputFmt != output.FormatTable {
				return output.Write(cmd.OutOrStdout(), rt().outputFmt, creds)
			}
			credDir := dir
			if credDir == "" {
				credDir, err = defaultBuilderCredDir(args[0])
				if err != nil {
					return err
				}
			}
			paths, err := writeBuilderCredentials(credDir, creds)
			if err != nil {
				return err
			}
			printBuilderCredentialsTable(cmd.OutOrStdout(), creds, paths)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Directory to write credential files (default: config dir/builders/<name>)")
	return cmd
}

func builderConnectCmd() *cobra.Command {
	var dir, buildxName string
	var setDefault, force bool
	cmd := &cobra.Command{
		Use:   "connect [name]",
		Short: "Configure local docker buildx for a remote exposed builder",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireJWTSession()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			return connectBuilder(cmd.Context(), org, args[0], connectBuilderOpts{
				dir:        dir,
				buildxName: buildxName,
				setDefault: setDefault,
				force:      force,
				wait:       true,
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Directory to write credential files (default: config dir/builders/<name>)")
	cmd.Flags().StringVar(&buildxName, "buildx-name", "", "Local buildx builder name (default: builderhub-<name>)")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Set the buildx builder as default")
	cmd.Flags().BoolVar(&force, "force", false, "Remove existing buildx builder with the same name before creating")
	return cmd
}
