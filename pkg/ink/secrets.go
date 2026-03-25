package ink

import "context"

const setSecretsMutation = `mutation($input: SetSecretsInput!) {
  serviceSetSecrets(input: $input) { serviceId name status }
}`

const deleteSecretsMutation = `mutation($input: DeleteSecretsInput!) {
  serviceDeleteSecrets(input: $input) { serviceId name status }
}`

// SetSecrets sets environment variables on a service. Existing vars with
// the same key are overwritten; other vars are preserved unless Replace is true.
// Triggers a redeployment.
func (c *Client) SetSecrets(ctx context.Context, input SetSecretsInput) error {
	return c.doGraphQL(ctx, setSecretsMutation, map[string]any{"input": input}, nil)
}

// DeleteSecrets removes the specified environment variable keys from a service.
// Triggers a redeployment.
func (c *Client) DeleteSecrets(ctx context.Context, input DeleteSecretsInput) error {
	return c.doGraphQL(ctx, deleteSecretsMutation, map[string]any{"input": input}, nil)
}
