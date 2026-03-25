package ink

import "context"

const execURLQuery = `query($serviceId: ID!) {
  serviceExecUrl(serviceId: $serviceId) { url token serviceId }
}`

const execQuery = `query($serviceId: ID!, $command: String!) {
  serviceExec(serviceId: $serviceId, command: $command) { exitCode stdout stderr }
}`

// ExecURL obtains a short-lived WebSocket URL and token for an interactive
// shell session. The token is valid for 120 seconds. Used by the exec
// sub-package to establish sessions.
func (c *Client) ExecURL(ctx context.Context, serviceID string) (*ExecSession, error) {
	var resp struct {
		ServiceExecUrl ExecSession `json:"serviceExecUrl"`
	}
	err := c.doGraphQL(ctx, execURLQuery, map[string]any{"serviceId": serviceID}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.ServiceExecUrl, nil
}

// Exec runs a command in a running service container and returns the output.
// Maximum 30 second timeout and 1 MiB output.
func (c *Client) Exec(ctx context.Context, serviceID, command string) (*ExecResult, error) {
	var resp struct {
		ServiceExec ExecResult `json:"serviceExec"`
	}
	vars := map[string]any{"serviceId": serviceID, "command": command}
	err := c.doGraphQL(ctx, execQuery, vars, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.ServiceExec, nil
}
