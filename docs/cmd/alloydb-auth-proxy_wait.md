## alloydb-auth-proxy wait

Wait for another Proxy process to start

### Synopsis


  Sometimes it is necessary to wait for the Proxy to start.

  To help ensure the Proxy is up and ready, the Proxy includes a wait
  subcommand with an optional --max flag to set the maximum time to wait.

  Invoke the wait command, like this:

  ./alloydb-auth-proxy wait

  By default, the Proxy will wait up to the maximum time for the startup
  endpoint to respond. The wait command requires that the Proxy be started in
  another process with the HTTP health check enabled. If an alternate health
  check port or address is used, as in:

  ./alloydb-auth-proxy <INSTANCE_URI> \
    --http-address 0.0.0.0 \
    --http-port 9191

  Then the wait command must also be told to use the same custom values:

  ./alloydb-auth-proxy wait \
    --http-address 0.0.0.0 \
    --http-port 9191

  By default the wait command will wait 30 seconds. To alter this value,
  use:

  ./alloydb-auth-proxy wait --max 10s


```
alloydb-auth-proxy wait [flags]
```

### Options

```
  -h, --help           help for wait
  -m, --max duration   maximum amount of time to wait for startup (default 30s)
```

### Options inherited from parent commands

```
      --http-address string   Address for Prometheus and health check server (default "localhost")
      --http-port string      Port for the Prometheus server to use (default "9090")
      --quiet                 Log error messages only
```

### SEE ALSO

* [alloydb-auth-proxy](alloydb-auth-proxy.md)	 - alloydb-auth-proxy provides a secure way to authorize connections to AlloyDB.

