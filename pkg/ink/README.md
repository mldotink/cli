# Ink Go SDK

Go client library for the [Ink](https://ml.ink) cloud platform.

## Installation

```bash
go get github.com/mldotink/cli/pkg/ink
```

## Usage

### Create a client

```go
import "github.com/mldotink/cli/pkg/ink"

client := ink.NewClient(ink.Config{
    APIKey: "dk_live_...", // https://ml.ink/account/api-keys
})
```

### Deploy a service

```go
result, err := client.CreateService(ctx, ink.CreateServiceInput{
    Name:   "my-api",
    Source: "image",
    Image:  "nginx:latest",
    Memory: "256Mi",
    VCPUs:  "0.25",
})
fmt.Println(result.ServiceID, result.Status)
```

### Get service status

```go
svc, err := client.GetService(ctx, serviceID)
fmt.Println(svc.Name, svc.Status)
```

### Run a command

```go
result, err := client.Exec(ctx, serviceID, "ls -la /app")
fmt.Println(result.Stdout)
fmt.Println("exit code:", result.ExitCode)
```

### Delete a service

```go
result, err := client.DeleteService(ctx, ink.DeleteServiceInput{
    ServiceID: serviceID,
})
fmt.Println(result.Message)
```

### Interactive shell session

```go
import "github.com/mldotink/cli/pkg/ink/exec"

session, err := exec.Dial(ctx, client, serviceID)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// Wire to stdin/stdout
go io.Copy(os.Stdout, session.Stdout())
go io.Copy(os.Stdout, session.Stderr())
io.Copy(session.Stdin(), os.Stdin)
```

### Set environment variables

```go
err := client.SetSecrets(ctx, ink.SetSecretsInput{
    ServiceID: serviceID,
    EnvVars: []ink.EnvVar{
        {Key: "DATABASE_URL", Value: "postgres://..."},
        {Key: "API_KEY", Value: "secret"},
    },
})
```

## API Reference

### Client methods

| Method | Description |
|--------|-------------|
| `CreateService` | Deploy a new service |
| `DeleteService` | Permanently remove a service |
| `UpdateService` | Reconfigure and redeploy |
| `GetService` | Get full service details |
| `ListServices` | List all services in a workspace |
| `Exec` | Run a one-shot command (30s timeout) |
| `ExecURL` | Get WebSocket URL for interactive shell |
| `SetSecrets` | Set environment variables |
| `DeleteSecrets` | Remove environment variables |

### exec.Session methods

| Method | Description |
|--------|-------------|
| `Stdin()` | Writer for shell input |
| `Stdout()` | Reader for shell stdout |
| `Stderr()` | Reader for shell stderr |
| `Resize(w, h)` | Send terminal resize |
| `Wait()` | Block until session ends |
| `Close()` | Terminate the session |
