package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/builderhub/build-cli/internal/client"
	"github.com/builderhub/build-cli/internal/config"
	"github.com/builderhub/build-cli/internal/output"
)

var version = "dev"

type runtime struct {
	cfgFile      *config.File
	profileName  string
	apiURL       string
	organization string
	tokenFlag    string
	outputFmt    output.Format
	client       *client.Client
}

var currentRuntime *runtime

var (
	flagAPIURL       string
	flagProfile      string
	flagOrganization string
	flagToken        string
	flagOutput       string
)

var rootCmd = &cobra.Command{
	Use:           "builderhub",
	Short:         "BuilderHub platform CLI",
	Long:          "Command-line interface for the BuilderHub platform.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		rt, err := buildRuntime()
		if err != nil {
			return err
		}
		currentRuntime = rt
		return nil
	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "BuilderHub API base URL")
	rootCmd.PersistentFlags().StringVar(&flagProfile, "profile", "", "Configuration profile name")
	rootCmd.PersistentFlags().StringVarP(&flagOrganization, "organization", "o", "", "Default organization (namespace) for builder commands")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "Bearer token override (JWT or bh_ API key)")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "O", "table", "Output format: table, json, yaml")

	rootCmd.AddCommand(
		authCmd(),
		apiKeyCmd(),
		orgCmd(),
		builderCmd(),
		configCmd(),
		healthCmd(),
		versionCmd(),
		completionCmd(),
	)
}

func buildRuntime() (*runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	profileName := flagProfile
	if profileName == "" {
		profileName = cfg.CurrentProfile
	}
	profile, err := cfg.Profile(profileName)
	if err != nil {
		profile = config.Profile{APIURL: config.DefaultAPIURL}
	}
	fmtOut, err := output.ParseFormat(flagOutput)
	if err != nil {
		return nil, err
	}
	apiURL := config.ResolveAPIURL(flagAPIURL, profile.APIURL)
	org := flagOrganization
	if org == "" {
		org = profile.Organization
	}
	store := &configStore{file: cfg, profile: profileName}
	token := config.ResolveToken(flagToken, profile, false)
	return &runtime{
		cfgFile:      cfg,
		profileName:  profileName,
		apiURL:       apiURL,
		organization: org,
		tokenFlag:    flagToken,
		outputFmt:    fmtOut,
		client:       client.New(apiURL, store, token),
	}, nil
}

func rt() *runtime {
	if currentRuntime != nil {
		return currentRuntime
	}
	r, err := buildRuntime()
	if err != nil {
		panic(err)
	}
	return r
}

type configStore struct {
	file    *config.File
	profile string
}

func (s *configStore) Token() string {
	p, err := s.file.Profile(s.profile)
	if err != nil {
		return ""
	}
	if p.AccessToken != "" {
		return p.AccessToken
	}
	return p.APIKey
}

func (s *configStore) RefreshToken() string {
	p, err := s.file.Profile(s.profile)
	if err != nil {
		return ""
	}
	return p.RefreshToken
}

func (s *configStore) SetTokens(access, refresh string) error {
	err := s.file.UpdateProfile(s.profile, func(p *config.Profile) {
		if access != "" {
			p.AccessToken = access
		}
		if refresh != "" {
			p.RefreshToken = refresh
		}
	})
	if err != nil {
		return err
	}
	return config.Save(s.file)
}

func requireOrganization() (string, error) {
	if rt().organization == "" {
		return "", fmt.Errorf("organization is required (use --organization or builderhub config set organization <id>)")
	}
	return rt().organization, nil
}

func requireJWTSession() error {
	profile, _ := rt().cfgFile.Profile(rt().profileName)
	token := config.ResolveToken(rt().tokenFlag, profile, false)
	if config.IsAPIKey(token) {
		return fmt.Errorf("this command requires a JWT session; run `builderhub auth login` (API keys cannot manage API keys or profile)")
	}
	if token == "" {
		return fmt.Errorf("not authenticated; run `builderhub auth login` or set BUILDERHUB_TOKEN")
	}
	return nil
}

func confirmDestructive(cmd *cobra.Command, yes bool, prompt string) error {
	if yes {
		return nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("%s (re-run with --yes to confirm)", prompt)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N]: ", prompt)
	var answer string
	if _, err := fmt.Scanln(&answer); err != nil {
		return fmt.Errorf("confirmation required; use --yes")
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("aborted")
	}
	return nil
}

func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func saveConfig() error {
	return config.Save(rt().cfgFile)
}
