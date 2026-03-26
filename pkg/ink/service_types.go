package ink

// Service represents a deployed service on Ink.
type Service struct {
	ID                 string        `json:"id"`
	ProjectID          string        `json:"projectId"`
	Name               string        `json:"name"`
	Subdomain          string        `json:"subdomain"`
	Source             string        `json:"source"`
	Repo               string        `json:"repo"`
	Image              string        `json:"image"`
	Branch             string        `json:"branch"`
	Status             string        `json:"status"`
	ErrorMessage       string        `json:"errorMessage"`
	EnvVars            []EnvVar      `json:"envVars"`
	Ports              []ServicePort `json:"ports"`
	GitProvider        string        `json:"gitProvider"`
	CommitHash         string        `json:"commitHash"`
	Memory             string        `json:"memory"`
	VCPUs              string        `json:"vcpus"`
	CustomDomain       string        `json:"customDomain"`
	CustomDomainStatus string        `json:"customDomainStatus"`
	BuildPack          string        `json:"buildPack"`
	BuildCommand       string        `json:"buildCommand"`
	StartCommand       string        `json:"startCommand"`
	PublishDirectory   string        `json:"publishDirectory"`
	RootDirectory      string        `json:"rootDirectory"`
	DockerfilePath        string        `json:"dockerfilePath"`
	TimeoutDestroySeconds int           `json:"timeoutDestroySeconds"`
	CreatedAt             string        `json:"createdAt"`
	UpdatedAt             string        `json:"updatedAt"`
}

// EnvVar is a key-value environment variable.
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ServicePort describes a port exposed by a service.
type ServicePort struct {
	Name             string `json:"name"`
	Port             string `json:"port"`
	Protocol         string `json:"protocol"`
	Visibility       string `json:"visibility"`
	InternalEndpoint string `json:"internalEndpoint"`
	PublicEndpoint   string `json:"publicEndpoint"`
}

// VolumeSpec defines a persistent volume to attach to a service.
type VolumeSpec struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	SizeGi    int    `json:"sizeGi,omitempty"`
}

// ServicePortInput defines a port for service creation/update.
type ServicePortInput struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	Visibility string `json:"visibility"`
}

// CreateServiceInput defines the parameters for creating a new service.
type CreateServiceInput struct {
	Name             string             `json:"name,omitempty"`
	Subdomain        string             `json:"subdomain,omitempty"`
	Source           string             `json:"source,omitempty"`
	Repo             string             `json:"repo,omitempty"`
	Image            string             `json:"image,omitempty"`
	Host             string             `json:"host,omitempty"`
	Branch           string             `json:"branch,omitempty"`
	Project          string             `json:"project,omitempty"`
	WorkspaceSlug    string             `json:"workspaceSlug,omitempty"`
	BuildPack        string             `json:"buildPack,omitempty"`
	Ports            []ServicePortInput `json:"ports,omitempty"`
	EnvVars          []EnvVar           `json:"envVars,omitempty"`
	Memory           string             `json:"memory,omitempty"`
	VCPUs            string             `json:"vcpus,omitempty"`
	BuildCommand     string             `json:"buildCommand,omitempty"`
	StartCommand     string             `json:"startCommand,omitempty"`
	PublishDirectory string             `json:"publishDirectory,omitempty"`
	RootDirectory    string             `json:"rootDirectory,omitempty"`
	DockerfilePath        string             `json:"dockerfilePath,omitempty"`
	Volumes               []VolumeSpec       `json:"volumes,omitempty"`
	TimeoutDestroySeconds int                `json:"timeoutDestroySeconds,omitempty"`
}

// CreateServiceResult is the result of creating a service.
type CreateServiceResult struct {
	ServiceID string        `json:"serviceId"`
	Name      string        `json:"name"`
	Status    string        `json:"status"`
	Repo      string        `json:"repo"`
	Ports     []ServicePort `json:"ports"`
}

// DeleteServiceInput identifies a service to delete.
type DeleteServiceInput struct {
	Name          string `json:"name,omitempty"`
	ServiceID     string `json:"serviceId,omitempty"`
	Project       string `json:"project,omitempty"`
	ProjectID     string `json:"projectId,omitempty"`
	WorkspaceSlug string `json:"workspaceSlug,omitempty"`
}

// DeleteServiceResult is the result of deleting a service.
type DeleteServiceResult struct {
	ServiceID string `json:"serviceId"`
	Name      string `json:"name"`
	Message   string `json:"message"`
}

// UpdateServiceInput defines the parameters for updating an existing service.
// Only non-nil pointer fields are sent to the API.
type UpdateServiceInput struct {
	Name             string             `json:"name,omitempty"`
	ServiceID        string             `json:"serviceId,omitempty"`
	Project          string             `json:"project,omitempty"`
	ProjectID        string             `json:"projectId,omitempty"`
	WorkspaceSlug    string             `json:"workspaceSlug,omitempty"`
	Source           *string            `json:"source,omitempty"`
	Image            *string            `json:"image,omitempty"`
	Repo             *string            `json:"repo,omitempty"`
	Host             *string            `json:"host,omitempty"`
	Branch           *string            `json:"branch,omitempty"`
	BuildPack        *string            `json:"buildPack,omitempty"`
	Memory           *string            `json:"memory,omitempty"`
	VCPUs            *string            `json:"vcpus,omitempty"`
	Ports            []ServicePortInput `json:"ports,omitempty"`
	EnvVars          []EnvVar           `json:"envVars,omitempty"`
	BuildCommand     *string            `json:"buildCommand,omitempty"`
	StartCommand     *string            `json:"startCommand,omitempty"`
	PublishDirectory *string            `json:"publishDirectory,omitempty"`
	RootDirectory    *string            `json:"rootDirectory,omitempty"`
	DockerfilePath        *string            `json:"dockerfilePath,omitempty"`
	Volumes               []VolumeSpec       `json:"volumes,omitempty"`
	TimeoutDestroySeconds *int               `json:"timeoutDestroySeconds,omitempty"`
}

// UpdateServiceResult is the result of updating a service.
type UpdateServiceResult struct {
	ServiceID string `json:"serviceId"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

// ExecSession contains the connection details for an interactive shell session.
type ExecSession struct {
	URL       string `json:"url"`
	Token     string `json:"token"`
	ServiceID string `json:"serviceId"`
}

// ExecResult is the output of a one-shot command execution.
type ExecResult struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// SetSecretsInput defines env vars to set on a service.
type SetSecretsInput struct {
	Name          string   `json:"name,omitempty"`
	ServiceID     string   `json:"serviceId,omitempty"`
	Project       string   `json:"project,omitempty"`
	ProjectID     string   `json:"projectId,omitempty"`
	WorkspaceSlug string   `json:"workspaceSlug,omitempty"`
	EnvVars       []EnvVar `json:"envVars"`
	Replace       bool     `json:"replace,omitempty"`
}

// DeleteSecretsInput defines env var keys to remove from a service.
type DeleteSecretsInput struct {
	Name          string   `json:"name,omitempty"`
	ServiceID     string   `json:"serviceId,omitempty"`
	Project       string   `json:"project,omitempty"`
	ProjectID     string   `json:"projectId,omitempty"`
	WorkspaceSlug string   `json:"workspaceSlug,omitempty"`
	Keys          []string `json:"keys"`
}
