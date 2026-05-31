package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/builderhub/build-cli/internal/client"
	"github.com/builderhub/build-cli/internal/output"
)

func templateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"templates"},
		Short:   "Manage builder templates",
	}
	cmd.AddCommand(
		templateListCmd(),
		templateGetCmd(),
		templateCreateCmd(),
		templateDeleteCmd(),
	)
	return cmd
}

func templateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List templates in an organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			templates, err := rt().client.ListTemplates(cmd.Context(), org)
			if err != nil {
				return err
			}
			return output.PrintTemplates(cmd.OutOrStdout(), rt().outputFmt, templates)
		},
	}
}

func templateGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [name]",
		Short: "Get a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			tpl, err := rt().client.GetTemplate(cmd.Context(), org, args[0])
			if err != nil {
				return err
			}
			return output.PrintTemplate(cmd.OutOrStdout(), rt().outputFmt, tpl)
		},
	}
}

type templateSpecFlags struct {
	buildkitImage string
	rootless      bool
	arch          string
	cacheType     string
	cacheSize     string
}

func (f templateSpecFlags) spec() client.TemplateSpec {
	spec := client.TemplateSpec{
		BuildkitImage: f.buildkitImage,
		Rootless:      f.rootless,
		Arch:          f.arch,
	}
	if f.cacheType != "" || f.cacheSize != "" {
		spec.CacheConfig = &client.CacheConfig{
			Type: f.cacheType,
		}
		if f.cacheType == "pvc" && f.cacheSize != "" {
			spec.CacheConfig.PVC = &struct {
				Size             string   `json:"size"`
				StorageClassName string   `json:"storage_class_name,omitempty"`
				AccessModes      []string `json:"access_modes,omitempty"`
			}{
				Size: f.cacheSize,
			}
		}
	}
	return spec
}

func addTemplateSpecFlags(cmd *cobra.Command, f *templateSpecFlags) {
	cmd.Flags().StringVar(&f.buildkitImage, "image", "moby/buildkit:master-rootless", "BuildKit image")
	cmd.Flags().BoolVar(&f.rootless, "rootless", true, "Run rootless")
	cmd.Flags().StringVar(&f.arch, "arch", "", "Arch (amd64/arm64)")
	cmd.Flags().StringVar(&f.cacheType, "cache-type", "pvc", "Cache type: pvc, none, s3")
	cmd.Flags().StringVar(&f.cacheSize, "cache-size", "25Gi", "Cache size (for pvc)")
}

func templateCreateCmd() *cobra.Command {
	var specFlags templateSpecFlags
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			spec := specFlags.spec()
			tpl, err := rt().client.CreateTemplate(cmd.Context(), org, args[0], spec)
			if err != nil {
				return err
			}
			return output.PrintTemplate(cmd.OutOrStdout(), rt().outputFmt, tpl)
		},
	}
	addTemplateSpecFlags(cmd, &specFlags)
	return cmd
}

func templateDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := requireOrganization()
			if err != nil {
				return err
			}
			if err := confirmDestructive(cmd, yes, fmt.Sprintf("Delete template %q?", args[0])); err != nil {
				return err
			}
			if err := rt().client.DeleteTemplate(cmd.Context(), org, args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Template deleted")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
