package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Organization struct {
	ID              string
	Name            string
	Slug            string
	Plan            string
	CreatedAt       int64
	BuilderCount    int32
	TotalMinutes    int64
	MonthlyMinutes  int64
	MemberCount     int32
}

type OrganizationMember struct {
	UserID   string
	Email    string
	Name     string
	Role     string
	JoinedAt int64
}

func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, "/v1/organizations", nil, &resp); err != nil {
		return nil, err
	}
	items, _ := resp["organizations"].([]any)
	out := make([]Organization, 0, len(items))
	for _, item := range items {
		raw, _ := item.(map[string]any)
		out = append(out, mapOrganization(raw))
	}
	return out, nil
}

func (c *Client) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, "/v1/organizations/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	orgRaw, _ := resp["organization"].(map[string]any)
	org := mapOrganization(orgRaw)
	return &org, nil
}

func (c *Client) CreateOrganization(ctx context.Context, name, slug, plan string) (*Organization, error) {
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPost, "/v1/organizations", map[string]string{
		"name": name,
		"slug": slug,
		"plan": plan,
	}, &resp); err != nil {
		return nil, err
	}
	orgRaw, _ := resp["organization"].(map[string]any)
	org := mapOrganization(orgRaw)
	return &org, nil
}

func (c *Client) UpdateOrganization(ctx context.Context, id string, name, slug, plan string) (*Organization, error) {
	body := map[string]string{"id": id}
	if name != "" {
		body["name"] = name
	}
	if slug != "" {
		body["slug"] = slug
	}
	if plan != "" {
		body["plan"] = plan
	}
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPatch, "/v1/organizations/"+url.PathEscape(id), body, &resp); err != nil {
		return nil, err
	}
	orgRaw, _ := resp["organization"].(map[string]any)
	org := mapOrganization(orgRaw)
	return &org, nil
}

func (c *Client) DeleteOrganization(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/v1/organizations/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ListOrganizationMembers(ctx context.Context, orgID string) ([]OrganizationMember, error) {
	path := fmt.Sprintf("/v1/organizations/%s/members", url.PathEscape(orgID))
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	items, _ := resp["members"].([]any)
	out := make([]OrganizationMember, 0, len(items))
	for _, item := range items {
		raw, _ := item.(map[string]any)
		out = append(out, mapOrganizationMember(raw))
	}
	return out, nil
}

func mapOrganization(raw map[string]any) Organization {
	if raw == nil {
		return Organization{}
	}
	return Organization{
		ID:             stringField(raw, "id"),
		Name:           stringField(raw, "name"),
		Slug:           stringField(raw, "slug"),
		Plan:           stringField(raw, "plan"),
		CreatedAt:      intField(raw, "created_at", "createdAt"),
		BuilderCount:   int32(intField(raw, "builder_count", "builderCount")),
		TotalMinutes:   intField(raw, "total_minutes", "totalMinutes"),
		MonthlyMinutes: intField(raw, "monthly_minutes", "monthlyMinutes"),
		MemberCount:    int32(intField(raw, "member_count", "memberCount")),
	}
}

func mapOrganizationMember(raw map[string]any) OrganizationMember {
	if raw == nil {
		return OrganizationMember{}
	}
	return OrganizationMember{
		UserID:   stringField(raw, "user_id", "userId"),
		Email:    stringField(raw, "email"),
		Name:     stringField(raw, "name"),
		Role:     stringField(raw, "role"),
		JoinedAt: intField(raw, "joined_at", "joinedAt"),
	}
}
