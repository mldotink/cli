package ink

import (
	"context"
	"fmt"
)

const serviceCreateMutation = `mutation($input: CreateServiceInput!) {
  serviceCreate(input: $input) {
    serviceId name status repo
    ports { name port protocol visibility internalEndpoint publicEndpoint }
  }
}`

const serviceDeleteMutation = `mutation($name: String, $serviceId: ID, $project: String, $projectId: ID, $ws: String) {
  serviceDelete(name: $name, serviceId: $serviceId, project: $project, projectId: $projectId, workspaceSlug: $ws) {
    serviceId name message
  }
}`

const serviceUpdateMutation = `mutation($input: UpdateServiceInput!) {
  serviceUpdate(input: $input) { serviceId name status }
}`

const serviceGetQuery = `query($id: ID!) {
  serviceGet(id: $id) {
    id projectId name subdomain source repo image branch status errorMessage
    envVars { key value }
    ports { name port protocol visibility internalEndpoint publicEndpoint }
    gitProvider commitHash memory vcpus customDomain customDomainStatus
    buildPack buildCommand startCommand publishDirectory rootDirectory dockerfilePath
    destroyTimeoutSeconds createdAt updatedAt
  }
}`

const serviceListQuery = `query($ws: String, $proj: String) {
  serviceList(workspaceSlug: $ws, projectSlug: $proj) {
    nodes {
      id projectId name subdomain source repo image branch status errorMessage
      envVars { key value }
      ports { name port protocol visibility internalEndpoint publicEndpoint }
      gitProvider commitHash memory vcpus customDomain customDomainStatus
      buildPack buildCommand startCommand publishDirectory rootDirectory dockerfilePath
      destroyTimeoutSeconds createdAt updatedAt
    }
  }
}`

// CreateService deploys a new service. The returned status is typically "queued"
// — poll GetService to track build/deploy progress.
func (c *Client) CreateService(ctx context.Context, input CreateServiceInput) (*CreateServiceResult, error) {
	var resp struct {
		ServiceCreate CreateServiceResult `json:"serviceCreate"`
	}
	err := c.doGraphQL(ctx, serviceCreateMutation, map[string]any{"input": input}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.ServiceCreate, nil
}

// DeleteService permanently removes a service and tears down its resources.
func (c *Client) DeleteService(ctx context.Context, input DeleteServiceInput) (*DeleteServiceResult, error) {
	var resp struct {
		ServiceDelete DeleteServiceResult `json:"serviceDelete"`
	}
	vars := map[string]any{}
	if input.Name != "" {
		vars["name"] = input.Name
	}
	if input.ServiceID != "" {
		vars["serviceId"] = input.ServiceID
	}
	if input.Project != "" {
		vars["project"] = input.Project
	}
	if input.ProjectID != "" {
		vars["projectId"] = input.ProjectID
	}
	if input.WorkspaceSlug != "" {
		vars["ws"] = input.WorkspaceSlug
	}
	err := c.doGraphQL(ctx, serviceDeleteMutation, vars, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.ServiceDelete, nil
}

// UpdateService reconfigures a service and triggers a redeployment.
func (c *Client) UpdateService(ctx context.Context, input UpdateServiceInput) (*UpdateServiceResult, error) {
	var resp struct {
		ServiceUpdate UpdateServiceResult `json:"serviceUpdate"`
	}
	err := c.doGraphQL(ctx, serviceUpdateMutation, map[string]any{"input": input}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.ServiceUpdate, nil
}

// GetService returns full details for a single service.
func (c *Client) GetService(ctx context.Context, id string) (*Service, error) {
	var resp struct {
		ServiceGet *Service `json:"serviceGet"`
	}
	err := c.doGraphQL(ctx, serviceGetQuery, map[string]any{"id": id}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.ServiceGet == nil {
		return nil, fmt.Errorf("ink: service %q not found", id)
	}
	return resp.ServiceGet, nil
}

// ListServices returns all services in a workspace, optionally filtered by project.
func (c *Client) ListServices(ctx context.Context, workspaceSlug, projectSlug string) ([]Service, error) {
	var resp struct {
		ServiceList struct {
			Nodes []Service `json:"nodes"`
		} `json:"serviceList"`
	}
	vars := map[string]any{}
	if workspaceSlug != "" {
		vars["ws"] = workspaceSlug
	}
	if projectSlug != "" {
		vars["proj"] = projectSlug
	}
	err := c.doGraphQL(ctx, serviceListQuery, vars, &resp)
	if err != nil {
		return nil, err
	}
	return resp.ServiceList.Nodes, nil
}
