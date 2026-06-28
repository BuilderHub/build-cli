package buildx

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	dockerRun = func(ctx context.Context, args ...string) error {
		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	dockerCombinedOutput = func(ctx context.Context, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	}
	dockerLookPath = exec.LookPath
)

type RemoteBuilderOpts struct {
	Name       string
	CACert     string
	ClientCert string
	ClientKey  string
	ServerName string
	Endpoint   string
	Force      bool
}

func CreateRemoteBuilder(ctx context.Context, opts RemoteBuilderOpts) error {
	if err := checkDocker(); err != nil {
		return err
	}
	if opts.Force {
		_ = RemoveBuilder(ctx, opts.Name)
	}
	args := []string{
		"buildx", "create",
		"--name", opts.Name,
		"--driver", "remote",
		"--driver-opt", "cacert=" + opts.CACert,
		"--driver-opt", "cert=" + opts.ClientCert,
		"--driver-opt", "key=" + opts.ClientKey,
		"--driver-opt", "servername=" + opts.ServerName,
		opts.Endpoint,
	}
	if err := dockerRun(ctx, args...); err != nil {
		return fmt.Errorf("docker buildx create: %w", err)
	}
	return nil
}

func UseBuilder(ctx context.Context, name string) error {
	if err := checkDocker(); err != nil {
		return err
	}
	if err := dockerRun(ctx, "buildx", "use", name); err != nil {
		return fmt.Errorf("docker buildx use: %w", err)
	}
	return nil
}

func RemoveBuilder(ctx context.Context, name string) error {
	out, err := dockerCombinedOutput(ctx, "buildx", "rm", name)
	if err != nil {
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "not found") || strings.Contains(msg, "does not exist") {
			return nil
		}
		return fmt.Errorf("docker buildx rm: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func checkDocker() error {
	if _, err := dockerLookPath("docker"); err != nil {
		return fmt.Errorf("docker not found in PATH; install Docker to use buildx integration")
	}
	return nil
}
