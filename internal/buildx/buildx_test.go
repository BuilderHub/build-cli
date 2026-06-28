package buildx

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCreateRemoteBuilder(t *testing.T) {
	var gotArgs []string
	dockerLookPath = func(string) (string, error) { return "/usr/bin/docker", nil }
	dockerRun = func(ctx context.Context, args ...string) error {
		gotArgs = args
		return nil
	}
	t.Cleanup(resetDockerMocks)

	if err := CreateRemoteBuilder(context.Background(), RemoteBuilderOpts{
		Name:       "builderhub-b1",
		CACert:     "/tmp/ca.pem",
		ClientCert: "/tmp/cert.pem",
		ClientKey:  "/tmp/key.pem",
		ServerName: "b1.example.com",
		Endpoint:   "tcp://b1.example.com:443",
	}); err != nil {
		t.Fatalf("CreateRemoteBuilder: %v", err)
	}
	joined := strings.Join(gotArgs, " ")
	for _, want := range []string{
		"buildx create",
		"--name builderhub-b1",
		"--driver remote",
		"cacert=/tmp/ca.pem",
		"cert=/tmp/cert.pem",
		"key=/tmp/key.pem",
		"servername=b1.example.com",
		"tcp://b1.example.com:443",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q: %s", want, joined)
		}
	}
}

func TestCreateRemoteBuilderForce(t *testing.T) {
	var runs [][]string
	dockerLookPath = func(string) (string, error) { return "/usr/bin/docker", nil }
	dockerRun = func(ctx context.Context, args ...string) error {
		runs = append(runs, args)
		return nil
	}
	dockerCombinedOutput = func(ctx context.Context, args ...string) ([]byte, error) {
		return nil, nil
	}
	t.Cleanup(resetDockerMocks)

	if err := CreateRemoteBuilder(context.Background(), RemoteBuilderOpts{
		Name:       "b1",
		CACert:     "/ca.pem",
		ClientCert: "/cert.pem",
		ClientKey:  "/key.pem",
		ServerName: "host",
		Endpoint:   "tcp://host:443",
		Force:      true,
	}); err != nil {
		t.Fatalf("CreateRemoteBuilder: %v", err)
	}
	if len(runs) != 1 || runs[0][0] != "buildx" || runs[0][1] != "create" {
		t.Fatalf("create run = %v", runs)
	}
}

func TestRemoveBuilderIgnoresNotFound(t *testing.T) {
	dockerCombinedOutput = func(ctx context.Context, args ...string) ([]byte, error) {
		return []byte("builder not found"), errors.New("exit status 1")
	}
	t.Cleanup(resetDockerMocks)

	if err := RemoveBuilder(context.Background(), "missing"); err != nil {
		t.Fatalf("RemoveBuilder: %v", err)
	}
}

func resetDockerMocks() {
	dockerLookPath = exec.LookPath
	dockerRun = func(ctx context.Context, args ...string) error {
		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	dockerCombinedOutput = func(ctx context.Context, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	}
}
