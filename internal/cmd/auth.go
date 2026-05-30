package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/builderhub/build-cli/internal/config"
	"github.com/builderhub/build-cli/internal/output"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate and manage account session",
	}
	cmd.AddCommand(authLoginCmd(), authRegisterCmd(), authLogoutCmd(), authWhoamiCmd(), authRefreshCmd())
	return cmd
}

func authLoginCmd() *cobra.Command {
	var email, password string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with email and password",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				fmt.Fprint(cmd.ErrOrStderr(), "Email: ")
				if _, err := fmt.Scanln(&email); err != nil {
					return fmt.Errorf("email required")
				}
			}
			if password == "" {
				var err error
				password, err = readPassword("Password: ")
				if err != nil {
					return err
				}
			}
			result, err := rt().client.Login(cmd.Context(), email, password)
			if err != nil {
				return err
			}
			if err := rt().cfgFile.UpdateProfile(rt().profileName, func(p *config.Profile) {
				p.AccessToken = result.AccessToken
				p.RefreshToken = result.RefreshToken
				p.APIKey = ""
			}); err != nil {
				return err
			}
			if err := saveConfig(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s (%s)\n", result.User.Name, result.User.Email)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Account email")
	cmd.Flags().StringVar(&password, "password", "", "Account password")
	return cmd
}

func authRegisterCmd() *cobra.Command {
	var email, password, name string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || password == "" || name == "" {
				return fmt.Errorf("--email, --password, and --name are required")
			}
			result, err := rt().client.Register(cmd.Context(), email, password, name)
			if err != nil {
				return err
			}
			if err := rt().cfgFile.UpdateProfile(rt().profileName, func(p *config.Profile) {
				p.AccessToken = result.AccessToken
				p.RefreshToken = result.RefreshToken
				p.APIKey = ""
			}); err != nil {
				return err
			}
			if err := saveConfig(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Registered and logged in as %s\n", result.User.Email)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Account email")
	cmd.Flags().StringVar(&password, "password", "", "Account password")
	cmd.Flags().StringVar(&name, "name", "", "Display name")
	return cmd
}

func authLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt().cfgFile.UpdateProfile(rt().profileName, func(p *config.Profile) {
				p.AccessToken = ""
				p.RefreshToken = ""
				p.APIKey = ""
			}); err != nil {
				return err
			}
			if err := saveConfig(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out")
			return nil
		},
	}
}

func authWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current authenticated user",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireJWTSession()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := rt().client.GetMe(cmd.Context())
			if err != nil {
				return err
			}
			return output.PrintUser(cmd.OutOrStdout(), rt().outputFmt, user)
		},
	}
}

func authRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the access token",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := rt().cfgFile.Profile(rt().profileName)
			if err != nil {
				return err
			}
			if profile.RefreshToken == "" {
				return fmt.Errorf("no refresh token stored; run `builderhub auth login`")
			}
			access, _, err := rt().client.RefreshToken(cmd.Context(), profile.RefreshToken)
			if err != nil {
				return err
			}
			if err := rt().cfgFile.UpdateProfile(rt().profileName, func(p *config.Profile) {
				p.AccessToken = access
			}); err != nil {
				return err
			}
			if err := saveConfig(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Access token refreshed")
			return nil
		},
	}
}
