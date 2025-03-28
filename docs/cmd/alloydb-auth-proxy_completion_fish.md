## alloydb-auth-proxy completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	alloydb-auth-proxy completion fish | source

To load completions for every new session, execute once:

	alloydb-auth-proxy completion fish > ~/.config/fish/completions/alloydb-auth-proxy.fish

You will need to start a new shell for this setup to take effect.


```
alloydb-auth-proxy completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --http-address string   Address for Prometheus and health check server (default "localhost")
      --http-port string      Port for the Prometheus server to use (default "9090")
      --quiet                 Log error messages only
```

### SEE ALSO

* [alloydb-auth-proxy completion](alloydb-auth-proxy_completion.md)	 - Generate the autocompletion script for the specified shell

