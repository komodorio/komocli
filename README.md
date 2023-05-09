# komocli
CLI to interact with Komodor platform

## Port Forwarding 

You need to know agent ID, provide valid JWT token and specify target names for objects

Example:
```shell
# komocli port-forward <agentId> <namespace>/<podName>:podPort localPort
komocli port-forward d14846d5-211f-466a-8b1e-c823a2270a99 default/helm-dashboard-86f46bdd56-kvgj7:8080 9090 --jwt=eyJh...jAw
```

JWT token can be specified via env variable `KOMOCLI_JWT`
`KOMOCLI_WS_URL` is the base URL for env, defaults to `wss://app.komodor.com`, `KOMOCLI_DEV` flag would make it use query string param for JWT instead of cookie.
`--address` sets the bind address for forwarder

# Roadmap, Ideas, TODOs

- make sure --help is meaningful
- test when wrong agent ID
- test when agent is down
- test when agent shuts down mid-session
- test when container shuts down mid-session
- test when CLI shuts down mid-session
