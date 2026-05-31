package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type CacheConfig struct {
	Type string `json:"type"`
	PVC  *struct {
		Size             string   `json:"size"`
		StorageClassName string   `json:"storage_class_name,omitempty"`
		AccessModes      []string `json:"access_modes,omitempty"`
	} `json:"pvc,omitempty"`
	S3 *struct {
		Bucket   string `json:"bucket"`
		Region   string `json:"region,omitempty"`
		Endpoint string `json:"endpoint,omitempty"`
	} `json:"s3,omitempty"`
}

type ResourceRequirements struct {
	Limits  map[string]string `json:"limits,omitempty"`
	Requests map[string]string `json:"requests,omitempty"`
}

type TemplateSpec struct {
	BuildkitImage string                `json:"buildkit_image,omitempty"`
	Rootless      bool                  `json:"rootless,omitempty"`
	Arch          string                `json:"arch,omitempty"`
	CacheConfig   *CacheConfig          `json:"cache_config,omitempty"`
	Resources     *ResourceRequirements `json:"resources,omitempty"`
}

type Template struct {
	Namespace string
	Name      string
	Spec      TemplateSpec
}

func (c *Client) ListTemplates(ctx context.Context, namespace string) ([]Template, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/templates", url.PathEscape(namespace))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	items, _ := resp["templates"].([]any)
	out := make([]Template, 0, len(items))
	for _, item := range items {
		raw, _ := item.(map[string]any)
		out = append(out, mapTemplate(raw))
	}
	return out, nil
}

func (c *Client) GetTemplate(ctx context.Context, namespace, name string) (*Template, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/templates/%s", url.PathEscape(namespace), url.PathEscape(name))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	tplRaw, _ := resp["template"].(map[string]any)
	t := mapTemplate(tplRaw)
	return &t, nil
}

func (c *Client) CreateTemplate(ctx context.Context, namespace, name string, spec TemplateSpec) (*Template, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/templates", url.PathEscape(namespace))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPost, path, map[string]any{
		"namespace": namespace,
		"name":      name,
		"spec":      spec,
	}, &resp); err != nil {
		return nil, err
	}
	tplRaw, _ := resp["template"].(map[string]any)
	t := mapTemplate(tplRaw)
	return &t, nil
}

func (c *Client) UpdateTemplate(ctx context.Context, namespace, name string, spec TemplateSpec) (*Template, error) {
	path := fmt.Sprintf("/v1/namespaces/%s/templates/%s", url.PathEscape(namespace), url.PathEscape(name))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPatch, path, map[string]any{
		"namespace": namespace,
		"name":      name,
		"spec":      spec,
	}, &resp); err != nil {
		return nil, err
	}
	tplRaw, _ := resp["template"].(map[string]any)
	t := mapTemplate(tplRaw)
	return &t, nil
}

func (c *Client) DeleteTemplate(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("/v1/namespaces/%s/templates/%s", url.PathEscape(namespace), url.PathEscape(name))
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

func mapTemplate(raw map[string]any) Template {
	if raw == nil {
		return Template{}
	}
	specRaw, _ := raw["spec"].(map[string]any)
	return Template{
		Namespace: stringField(raw, "namespace"),
		Name:      stringField(raw, "name"),
		Spec:      mapTemplateSpec(specRaw),
	}
}

func mapTemplateSpec(raw map[string]any) TemplateSpec {
	if raw == nil {
		return TemplateSpec{}
	}
	spec := TemplateSpec{
		BuildkitImage: stringField(raw, "buildkit_image", "buildkitImage"),
		Rootless:      raw["rootless"] == true || stringField(raw, "rootless") == "true",
		Arch:          stringField(raw, "arch"),
	}
	if cc, ok := raw["cache_config"].(map[string]any); ok {
		spec.CacheConfig = mapCacheConfig(cc)
	} else if cc, ok := raw["cacheConfig"].(map[string]any); ok {
		spec.CacheConfig = mapCacheConfig(cc)
	}
	if res, ok := raw["resources"].(map[string]any); ok {
		spec.Resources = mapResources(res)
	}
	return spec
}

func mapCacheConfig(raw map[string]any) *CacheConfig {
	if raw == nil {
		return nil
	}
	cc := &CacheConfig{
		Type: stringField(raw, "type"),
	}
	if pvc, ok := raw["pvc"].(map[string]any); ok {
		cc.PVC = &struct {
			Size             string   `json:"size"`
			StorageClassName string   `json:"storage_class_name,omitempty"`
			AccessModes      []string `json:"access_modes,omitempty"`
		}{
			Size:             stringField(pvc, "size"),
			StorageClassName: stringField(pvc, "storage_class_name", "storageClassName"),
		}
		if am, ok := pvc["access_modes"].([]any); ok {
			for _, a := range am {
				cc.PVC.AccessModes = append(cc.PVC.AccessModes, fmt.Sprint(a))
			}
		}
	}
	// S3 omitted for brevity in basic CLI
	return cc
}

func mapResources(raw map[string]any) *ResourceRequirements {
	if raw == nil {
		return nil
	}
	rr := &ResourceRequirements{}
	if limits, ok := raw["limits"].(map[string]any); ok {
		rr.Limits = make(map[string]string)
		for k, v := range limits {
			rr.Limits[k] = fmt.Sprint(v)
		}
	}
	if reqs, ok := raw["requests"].(map[string]any); ok {
		rr.Requests = make(map[string]string)
		for k, v := range reqs {
			rr.Requests[k] = fmt.Sprint(v)
		}
	}
	return rr
}
