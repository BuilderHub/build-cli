package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/builderhub/build-cli/internal/buildx"
	"github.com/builderhub/build-cli/internal/client"
	"github.com/builderhub/build-cli/internal/config"
)

const (
	builderCredCAFile   = "ca.pem"
	builderCredCertFile = "client-cert.pem"
	builderCredKeyFile  = "client-key.pem"

	builderPollInterval = 5 * time.Second
	builderPollTimeout  = 3 * time.Minute
)

type builderCredentialPaths struct {
	Dir        string
	CA         string
	ClientCert string
	ClientKey  string
}

func defaultBuilderCredDir(name string) (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "builders", name), nil
}

func writeBuilderCredentials(dir string, creds *client.BuilderCredentials) (builderCredentialPaths, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return builderCredentialPaths{}, err
	}
	paths := builderCredentialPaths{
		Dir:        dir,
		CA:         filepath.Join(dir, builderCredCAFile),
		ClientCert: filepath.Join(dir, builderCredCertFile),
		ClientKey:  filepath.Join(dir, builderCredKeyFile),
	}
	if err := os.WriteFile(paths.CA, []byte(creds.CAPEM), 0o644); err != nil {
		return builderCredentialPaths{}, err
	}
	if err := os.WriteFile(paths.ClientCert, []byte(creds.ClientCertPEM), 0o644); err != nil {
		return builderCredentialPaths{}, err
	}
	if err := os.WriteFile(paths.ClientKey, []byte(creds.ClientKeyPEM), 0o600); err != nil {
		return builderCredentialPaths{}, err
	}
	return paths, nil
}

func defaultBuildxName(builderName string) string {
	return "builderhub-" + builderName
}

func waitForExposedBuilder(ctx context.Context, org, name string) (*client.Builder, error) {
	deadline := time.Now().Add(builderPollTimeout)
	for {
		builder, err := rt().client.GetBuilder(ctx, org, name)
		if err != nil {
			return nil, err
		}
		if builder.Status.ExternalEndpoint != "" && builder.Status.Phase == "Ready" {
			return builder, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for builder %q to become exposed and ready (external endpoint not available)", name)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(builderPollInterval):
		}
	}
}

type connectBuilderOpts struct {
	dir         string
	buildxName  string
	setDefault  bool
	force       bool
	wait        bool
}

func connectBuilder(ctx context.Context, org, name string, opts connectBuilderOpts) error {
	if opts.wait {
		if _, err := waitForExposedBuilder(ctx, org, name); err != nil {
			return err
		}
	}

	creds, err := rt().client.GenerateBuilderCredentials(ctx, org, name)
	if err != nil {
		return err
	}

	dir := opts.dir
	if dir == "" {
		dir, err = defaultBuilderCredDir(name)
		if err != nil {
			return err
		}
	}

	paths, err := writeBuilderCredentials(dir, creds)
	if err != nil {
		return err
	}

	buildxName := opts.buildxName
	if buildxName == "" {
		buildxName = defaultBuildxName(name)
	}

	if err := buildx.CreateRemoteBuilder(ctx, buildx.RemoteBuilderOpts{
		Name:       buildxName,
		CACert:     paths.CA,
		ClientCert: paths.ClientCert,
		ClientKey:  paths.ClientKey,
		ServerName: creds.ServerName,
		Endpoint:   creds.Endpoint,
		Force:      opts.force,
	}); err != nil {
		return err
	}

	if opts.setDefault {
		if err := buildx.UseBuilder(ctx, buildxName); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "Connected buildx builder %q\n", buildxName)
	fmt.Fprintf(os.Stdout, "Endpoint:    %s\n", creds.Endpoint)
	fmt.Fprintf(os.Stdout, "Server name: %s\n", creds.ServerName)
	fmt.Fprintf(os.Stdout, "Credentials: %s\n", paths.Dir)
	fmt.Fprintf(os.Stdout, "\nExample build:\n  docker buildx build --builder %s -t myimage --push .\n", buildxName)
	return nil
}

func printBuilderCredentialsTable(w io.Writer, creds *client.BuilderCredentials, paths builderCredentialPaths) {
	fmt.Fprintln(w, "New mTLS credentials generated. Previously issued credentials remain valid until they expire.")
	fmt.Fprintf(w, "Endpoint:    %s\n", creds.Endpoint)
	fmt.Fprintf(w, "Server name: %s\n", creds.ServerName)
	if creds.ExpiresAt > 0 {
		fmt.Fprintf(w, "Expires:     %s\n", time.Unix(creds.ExpiresAt, 0).UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(w, "CA:          %s\n", paths.CA)
	fmt.Fprintf(w, "Client cert: %s\n", paths.ClientCert)
	fmt.Fprintf(w, "Client key:  %s\n", paths.ClientKey)
}
