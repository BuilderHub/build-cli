package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type BuilderSpec struct {
	TemplateRef        string            `json:"template_ref,omitempty"`
	Mode               string            `json:"mode,omitempty"`
	Replicas           int32             `json:"replicas,omitempty"`
	IdleTimeoutSeconds int32             `json:"idle_timeout_seconds,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	Expose             *bool             `json:"expose,omitempty"`
}

type BuilderStatus struct {
	Endpoint         string
	ExternalEndpoint string
	NodePort         int32
	Phase            string
}

type BuilderCredentials struct {
	CAPEM         string
	ClientCertPEM string
	ClientKeyPEM  string
	Endpoint      string
	ServerName    string
	ExpiresAt     int64
}

type Builder struct {
	Namespace string
	Name      string
	Spec      BuilderSpec
	Status    BuilderStatus
}

func (c *Client) ListBuilders(ctx context.Context, namespace string) ([]Builder, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/builders", url.PathEscape(namespace))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	items, _ := resp["builders"].([]any)
	out := make([]Builder, 0, len(items))
	for _, item := range items {
		raw, _ := item.(map[string]any)
		out = append(out, mapBuilder(raw))
	}
	return out, nil
}

func (c *Client) GetBuilder(ctx context.Context, namespace, name string) (*Builder, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/builders/%s", url.PathEscape(namespace), url.PathEscape(name))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	builderRaw, _ := resp["builder"].(map[string]any)
	b := mapBuilder(builderRaw)
	return &b, nil
}

func (c *Client) CreateBuilder(ctx context.Context, namespace, name string, spec BuilderSpec) (*Builder, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/builders", url.PathEscape(namespace))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPost, path, map[string]any{
		"namespace": namespace,
		"name":      name,
		"spec":      spec,
	}, &resp); err != nil {
		return nil, err
	}
	builderRaw, _ := resp["builder"].(map[string]any)
	b := mapBuilder(builderRaw)
	return &b, nil
}

func (c *Client) UpdateBuilder(ctx context.Context, namespace, name string, spec BuilderSpec) (*Builder, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/builders/%s", url.PathEscape(namespace), url.PathEscape(name))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPatch, path, map[string]any{
		"namespace": namespace,
		"name":      name,
		"spec":      spec,
	}, &resp); err != nil {
		return nil, err
	}
	builderRaw, _ := resp["builder"].(map[string]any)
	b := mapBuilder(builderRaw)
	return &b, nil
}

func (c *Client) DeleteBuilder(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("/v1/namespaces/%s/builders/%s", url.PathEscape(namespace), url.PathEscape(name))
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) GenerateBuilderCredentials(ctx context.Context, namespace, name string) (*BuilderCredentials, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/builders/%s/credentials", url.PathEscape(namespace), url.PathEscape(name))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPost, path, map[string]any{
		"namespace": namespace,
		"name":      name,
	}, &resp); err != nil {
		return nil, err
	}
	return mapBuilderCredentials(resp), nil
}

func (c *Client) WakeBuilder(ctx context.Context, namespace, name string) (*Builder, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/builders/%s/wake", url.PathEscape(namespace), url.PathEscape(name))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPost, path, map[string]any{
		"namespace": namespace,
		"name":      name,
	}, &resp); err != nil {
		return nil, err
	}
	builderRaw, _ := resp["builder"].(map[string]any)
	b := mapBuilder(builderRaw)
	return &b, nil
}

func (c *Client) HealthCheck(ctx context.Context) (string, error) {
	var resp map[string]any
	if err := c.do(ctx, http.MethodGet, "/v1/health", nil, &resp, false); err != nil {
		return "", err
	}
	return stringField(resp, "status"), nil
}

func mapBuilder(raw map[string]any) Builder {
	if raw == nil {
		return Builder{}
	}
	specRaw, _ := raw["spec"].(map[string]any)
	statusRaw, _ := raw["status"].(map[string]any)
	return Builder{
		Namespace: stringField(raw, "namespace"),
		Name:      stringField(raw, "name"),
		Spec:      mapBuilderSpec(specRaw),
		Status:    mapBuilderStatus(statusRaw),
	}
}

func mapBuilderSpec(raw map[string]any) BuilderSpec {
	if raw == nil {
		return BuilderSpec{}
	}
	labels := map[string]string{}
	if labelsRaw, ok := raw["labels"].(map[string]any); ok {
		for k, v := range labelsRaw {
			labels[k] = fmt.Sprint(v)
		}
	}
	spec := BuilderSpec{
		TemplateRef:        stringField(raw, "template_ref", "templateRef"),
		Mode:               stringField(raw, "mode"),
		Replicas:           int32(intField(raw, "replicas")),
		IdleTimeoutSeconds: int32(intField(raw, "idle_timeout_seconds", "idleTimeoutSeconds")),
		Labels:             labels,
	}
	if expose, ok := boolField(raw, "expose"); ok {
		spec.Expose = &expose
	}
	return spec
}

func mapBuilderStatus(raw map[string]any) BuilderStatus {
	if raw == nil {
		return BuilderStatus{}
	}
	return BuilderStatus{
		Endpoint:         stringField(raw, "endpoint"),
		ExternalEndpoint: stringField(raw, "external_endpoint", "externalEndpoint"),
		NodePort:         int32(intField(raw, "node_port", "nodePort")),
		Phase:            stringField(raw, "phase", "Phase"),
	}
}

func mapBuilderCredentials(raw map[string]any) *BuilderCredentials {
	if raw == nil {
		return &BuilderCredentials{}
	}
	return &BuilderCredentials{
		CAPEM:         stringField(raw, "ca_pem", "caPem"),
		ClientCertPEM: stringField(raw, "client_cert_pem", "clientCertPem"),
		ClientKeyPEM:  stringField(raw, "client_key_pem", "clientKeyPem"),
		Endpoint:      stringField(raw, "endpoint"),
		ServerName:    stringField(raw, "server_name", "serverName"),
		ExpiresAt:     intField(raw, "expires_at", "expiresAt"),
	}
}

func boolField(raw map[string]any, keys ...string) (bool, bool) {
	for _, k := range keys {
		v, ok := raw[k]
		if !ok || v == nil {
			continue
		}
		switch b := v.(type) {
		case bool:
			return b, true
		case float64:
			return b != 0, true
		}
	}
	return false, false
}
