# komocli
CLI to interact with Komodor platform

## Port Forwarding

### Basic Usage

```shell
# Default (US region)
komocli port-forward pod/mypod 8888:5000 --namespace default --cluster my-cluster --token=...

# EU region
komocli port-forward pod/mypod 8888:5000 --namespace default --cluster my-cluster --token=... --region eu

# Custom WebSocket URL (advanced)
komocli port-forward pod/mypod 5000 --namespace default --cluster my-cluster --token=... --region wss://custom.komodor.com
```

### Authentication

JWT token can be specified via:
- Command line: `--token YOUR_TOKEN`
- Environment variable: `KOMOCLI_JWT`

### Region Configuration

The `--region` flag supports:
- **Predefined regions**: `us` (default), `eu`

### Environment Variables

- `KOMOCLI_JWT` - JWT authentication token (alternative to --token flag)
- `DEBUG` - Enable verbose logging

### Additional Options

- `--address` - Sets the bind address for forwarder (default: localhost)
- `--browser` - Automatically open forwarded address in browser
- `--timeout` - Timeout for operations (default: 5s)

# Roadmap, Ideas, TODOs

- make sure --help is meaningful
- test when wrong agent ID
- test when agent is down
- test when agent shuts down mid-session
- test when container shuts down mid-session
- test when CLI shuts down mid-session
