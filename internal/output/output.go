package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/builderhub/build-cli/internal/client"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unknown output format %q (use table, json, or yaml)", s)
	}
}

func Write(w io.Writer, format Format, v any) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case FormatYAML:
		data, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		return fmt.Errorf("unsupported structured output %q", format)
	}
}

func PrintUser(w io.Writer, format Format, u *client.User) error {
	if format != FormatTable {
		return Write(w, format, u)
	}
	fmt.Fprintf(w, "ID:        %s\n", u.ID)
	fmt.Fprintf(w, "Email:     %s\n", u.Email)
	fmt.Fprintf(w, "Name:      %s\n", u.Name)
	fmt.Fprintf(w, "Created:   %s\n", formatUnix(u.CreatedAt))
	return nil
}

func PrintOrganizations(w io.Writer, format Format, orgs []client.Organization) error {
	if format != FormatTable {
		return Write(w, format, orgs)
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSLUG\tPLAN\tBUILDERS\tMEMBERS\tCREATED")
	for _, o := range orgs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%s\n",
			o.ID, o.Slug, o.Plan, o.BuilderCount, o.MemberCount, formatUnix(o.CreatedAt))
	}
	return tw.Flush()
}

func PrintOrganization(w io.Writer, format Format, o *client.Organization) error {
	if format != FormatTable {
		return Write(w, format, o)
	}
	fmt.Fprintf(w, "ID:              %s\n", o.ID)
	fmt.Fprintf(w, "Name:            %s\n", o.Name)
	fmt.Fprintf(w, "Slug:            %s\n", o.Slug)
	fmt.Fprintf(w, "Plan:            %s\n", o.Plan)
	fmt.Fprintf(w, "Builders:        %d\n", o.BuilderCount)
	fmt.Fprintf(w, "Members:         %d\n", o.MemberCount)
	fmt.Fprintf(w, "Total minutes:   %d\n", o.TotalMinutes)
	fmt.Fprintf(w, "Monthly minutes: %d\n", o.MonthlyMinutes)
	fmt.Fprintf(w, "Created:         %s\n", formatUnix(o.CreatedAt))
	return nil
}

func PrintOrganizationMembers(w io.Writer, format Format, members []client.OrganizationMember) error {
	if format != FormatTable {
		return Write(w, format, members)
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "USER ID\tEMAIL\tNAME\tROLE\tJOINED")
	for _, m := range members {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			m.UserID, m.Email, m.Name, m.Role, formatUnix(m.JoinedAt))
	}
	return tw.Flush()
}

func PrintBuilders(w io.Writer, format Format, builders []client.Builder) error {
	if format != FormatTable {
		return Write(w, format, builders)
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tMODE\tPHASE\tENDPOINT\tREPLICAS")
	for _, b := range builders {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n",
			b.Name, b.Spec.Mode, b.Status.Phase, b.Status.Endpoint, b.Spec.Replicas)
	}
	return tw.Flush()
}

func PrintBuilder(w io.Writer, format Format, b *client.Builder) error {
	if format != FormatTable {
		return Write(w, format, b)
	}
	fmt.Fprintf(w, "Name:       %s\n", b.Name)
	fmt.Fprintf(w, "Namespace:  %s\n", b.Namespace)
	fmt.Fprintf(w, "Mode:       %s\n", b.Spec.Mode)
	fmt.Fprintf(w, "Replicas:   %d\n", b.Spec.Replicas)
	fmt.Fprintf(w, "Template:   %s\n", b.Spec.TemplateRef)
	fmt.Fprintf(w, "Idle (s):   %d\n", b.Spec.IdleTimeoutSeconds)
	fmt.Fprintf(w, "Phase:      %s\n", b.Status.Phase)
	fmt.Fprintf(w, "Endpoint:   %s\n", b.Status.Endpoint)
	fmt.Fprintf(w, "Node port:  %d\n", b.Status.NodePort)
	if len(b.Spec.Labels) > 0 {
		fmt.Fprintln(w, "Labels:")
		for k, v := range b.Spec.Labels {
			fmt.Fprintf(w, "  %s=%s\n", k, v)
		}
	}
	return nil
}

func PrintAPIKeys(w io.Writer, format Format, keys []client.UserAPIKey) error {
	if format != FormatTable {
		return Write(w, format, keys)
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tPREFIX\tSCOPES\tEXPIRES\tCREATED")
	for _, k := range keys {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			k.ID, k.Name, k.KeyPrefix, strings.Join(k.Scopes, ","),
			formatExpiry(k.ExpiresAt), formatUnix(k.CreatedAt))
	}
	return tw.Flush()
}

func PrintAPIKey(w io.Writer, format Format, key client.UserAPIKey) error {
	if format != FormatTable {
		return Write(w, format, key)
	}
	fmt.Fprintf(w, "ID:       %s\n", key.ID)
	fmt.Fprintf(w, "Name:     %s\n", key.Name)
	fmt.Fprintf(w, "Prefix:   %s\n", key.KeyPrefix)
	fmt.Fprintf(w, "Scopes:   %s\n", strings.Join(key.Scopes, ", "))
	fmt.Fprintf(w, "Created:  %s\n", formatUnix(key.CreatedAt))
	fmt.Fprintf(w, "Expires:  %s\n", formatExpiry(key.ExpiresAt))
	return nil
}

func formatUnix(ts int64) string {
	if ts <= 0 {
		return "-"
	}
	if ts > 1_000_000_000_000 {
		return time.UnixMilli(ts).UTC().Format(time.RFC3339)
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func formatExpiry(ts int64) string {
	if ts == 0 {
		return "never"
	}
	return formatUnix(ts)
}
