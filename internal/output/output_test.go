package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/builderhub/build-cli/internal/client"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		in      string
		want    Format
		wantErr bool
	}{
		{"", FormatTable, false},
		{"table", FormatTable, false},
		{"TABLE", FormatTable, false},
		{" json ", FormatJSON, false},
		{"yaml", FormatYAML, false},
		{"xml", "", true},
	}
	for _, tt := range tests {
		got, err := ParseFormat(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParseFormat(%q): expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseFormat(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseFormat(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWriteJSONAndYAML(t *testing.T) {
	payload := map[string]string{"name": "builderhub"}

	var jsonBuf bytes.Buffer
	if err := Write(&jsonBuf, FormatJSON, payload); err != nil {
		t.Fatalf("Write JSON: %v", err)
	}
	if !strings.Contains(jsonBuf.String(), `"name": "builderhub"`) {
		t.Fatalf("JSON output = %q", jsonBuf.String())
	}

	var yamlBuf bytes.Buffer
	if err := Write(&yamlBuf, FormatYAML, payload); err != nil {
		t.Fatalf("Write YAML: %v", err)
	}
	if !strings.Contains(yamlBuf.String(), "name: builderhub") {
		t.Fatalf("YAML output = %q", yamlBuf.String())
	}
}

func TestWriteUnsupportedFormat(t *testing.T) {
	err := Write(&bytes.Buffer{}, FormatTable, map[string]string{"x": "y"})
	if err == nil {
		t.Fatal("expected error for table format in Write")
	}
}

func TestPrintUserTable(t *testing.T) {
	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC).Unix()
	u := &client.User{ID: "u1", Email: "a@b.com", Name: "Alice", CreatedAt: ts}

	var buf bytes.Buffer
	if err := PrintUser(&buf, FormatTable, u); err != nil {
		t.Fatalf("PrintUser: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"ID:        u1", "Email:     a@b.com", "Name:      Alice", "2024-06-01T12:00:00Z"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintUserJSON(t *testing.T) {
	u := &client.User{ID: "u1", Email: "a@b.com", Name: "Alice"}
	var buf bytes.Buffer
	if err := PrintUser(&buf, FormatJSON, u); err != nil {
		t.Fatalf("PrintUser: %v", err)
	}
	if !strings.Contains(buf.String(), `"ID": "u1"`) {
		t.Fatalf("JSON output = %q", buf.String())
	}
}

func TestPrintBuildersTable(t *testing.T) {
	exposed := true
	builders := []client.Builder{{
		Name: "b1",
		Spec: client.BuilderSpec{Mode: "sleepy", Replicas: 2, Expose: &exposed},
		Status: client.BuilderStatus{Phase: "Ready", Endpoint: "10.0.0.1", ExternalEndpoint: "tcp://exposed.example.com:443"},
	}}

	var buf bytes.Buffer
	if err := PrintBuilders(&buf, FormatTable, builders); err != nil {
		t.Fatalf("PrintBuilders: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAME", "b1", "sleepy", "Ready", "yes", "tcp://exposed.example.com:443", "2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestFormatUnixViaPrintUser(t *testing.T) {
	tests := []struct {
		name      string
		createdAt int64
		want      string
	}{
		{"zero", 0, "Created:   -\n"},
		{"negative", -1, "Created:   -\n"},
		{"seconds", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC).Unix(), "2024-01-02T03:04:05Z"},
		{"millis", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC).UnixMilli(), "2024-01-02T03:04:05Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &client.User{ID: "u1", CreatedAt: tt.createdAt}
			var buf bytes.Buffer
			if err := PrintUser(&buf, FormatTable, u); err != nil {
				t.Fatalf("PrintUser: %v", err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Fatalf("output = %q, want substring %q", buf.String(), tt.want)
			}
		})
	}
}

func TestFormatExpiryViaPrintAPIKey(t *testing.T) {
	t.Run("never", func(t *testing.T) {
		key := client.UserAPIKey{ID: "k1", Name: "dev", ExpiresAt: 0}
		var buf bytes.Buffer
		if err := PrintAPIKey(&buf, FormatTable, key); err != nil {
			t.Fatalf("PrintAPIKey: %v", err)
		}
		if !strings.Contains(buf.String(), "Expires:  never") {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("timestamp", func(t *testing.T) {
		ts := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC).Unix()
		key := client.UserAPIKey{ID: "k1", Name: "dev", ExpiresAt: ts}
		var buf bytes.Buffer
		if err := PrintAPIKey(&buf, FormatTable, key); err != nil {
			t.Fatalf("PrintAPIKey: %v", err)
		}
		if !strings.Contains(buf.String(), "2025-03-15T00:00:00Z") {
			t.Fatalf("output = %q", buf.String())
		}
	})
}
