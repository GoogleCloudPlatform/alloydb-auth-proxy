## alloydb-auth-proxy

alloydb-auth-proxy provides a secure way to authorize connections to AlloyDB.

### Synopsis


Overview

  The AlloyDB Auth proxy is a utility for ensuring secure connections
  to your AlloyDB instances. It provides IAM authorization, allowing you
  to control who can connect to your instances through IAM permissions, and TLS
  1.3 encryption, without having to manage certificates.

  NOTE: The proxy does not configure the network. You MUST ensure the proxy
  can reach your AlloyDB instance, either by deploying it in a VPC that has
  access to your instance, or by ensuring a network path to the instance.

  For every provided instance connection name, the proxy creates:

  - a socket that mimics a database running locally, and
  - an encrypted connection using TLS 1.3 back to your AlloyDB instance.

  The proxy uses an ephemeral certificate to establish a secure connection to
  your AlloyDB instance. The proxy will refresh those certificates on an
  hourly basis. Existing client connections are unaffected by the refresh
  cycle.

Authentication

  The Proxy uses Application Default Credentials by default. Enable these
  credentials with gcloud:

      gcloud auth application-default login

  In Google-run environments, Application Default Credentials are already
  available and do not need to be retrieved.

  The Proxy will use the environment's IAM principal when authenticating to
  the backend. To use a specific set of credentials, use the
  --credentials-file flag, e.g.,

      ./alloydb-auth-proxy --credentials-file /path/to/key.json \
          projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE

  See the individual flags below, for more options.

Starting the Proxy

  To start the proxy, you will need your instance URI, which may be found in
  the AlloyDB instance overview page or by using gcloud with the following
  command:

      gcloud alpha alloydb instances describe INSTANCE_NAME \
          --region=REGION --cluster CLUSTER_NAME --format='value(name)'

  For example, if your instance URI is:

      projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE

  Starting the proxy will look like:

      ./alloydb-auth-proxy \
          projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE

  By default, the proxy will start a TCP listener on Postgres' default port
  5432. If multiple instances are specified which all use the same database
  engine, the first will be started on the default port and subsequent
  instances will be incremented from there (e.g., 5432, 5433, 5434, etc.) To
  disable this behavior, use the --port flag. All subsequent listeners will
  increment from the provided value.

  All socket listeners use the localhost network interface. To override this
  behavior, use the --address flag.

Instance Level Configuration

  The proxy supports overriding configuration on an instance-level with an
  optional query string syntax using the corresponding full flag name. The
  query string takes the form of a URL query string and should be appended to
  the instance URI, e.g.,

      'projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE?key1=value1'

  When using the optional query string syntax, quotes must wrap the instance
  connection name and query string to prevent conflicts with the shell. For
  example, to override the address and port for one instance but otherwise use
  the default behavior, use:

      ./alloydb-auth-proxy \
          'projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE1' \
          'projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE2?address=0.0.0.0&port=7000'

  When necessary, you may specify the full path to a Unix socket. Set the
  unix-socket-path query parameter to the absolute path of the Unix socket for
  the database instance. The parent directory of the unix-socket-path must
  exist when the proxy starts or else socket creation will fail. For Postgres
  instances, the proxy will ensure that the last path element is
  '.s.PGSQL.5432' appending it if necessary. For example,

      ./alloydb-auth-proxy \
          'projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE1?unix-socket-path=/path/to/socket'

Automatic IAM Authentication

  The Auth Proxy support Automatic IAM Authentication where the Proxy
  retrieves the environment's IAM principal's OAuth2 token and supplies it to
  the backend. When a client connects to the Proxy, there is no need to supply
  a database user password.

  To enable the feature, run:

      ./alloydb-auth-proxy \
          --auto-iam-authn \
          'projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE'

  In addition, Auto IAM AuthN may be enabled on a per-instance basis with the
  query string syntax described above.

      ./alloydb-auth-proxy \
          'projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE?auto-iam-authn=true'

Health checks

  When enabling the --health-check flag, the proxy will start an HTTP server
  on localhost with three endpoints:

  - /startup: Returns 200 status when the proxy has finished starting up.
  Otherwise returns 503 status.

  - /readiness: Returns 200 status when the proxy has started, has available
  connections if max connections have been set with the --max-connections
  flag, and when the proxy can connect to all registered instances. Otherwise,
  returns a 503 status.

  - /liveness: Always returns 200 status. If this endpoint is not responding,
  the proxy is in a bad state and should be restarted.

  To configure the address, use --http-address. To configure the port, use
  --http-port.

Service Account Impersonation

  The proxy supports service account impersonation with the
  --impersonate-service-account flag and matches gcloud's flag. When enabled,
  all API requests are made impersonating the supplied service account. The
  IAM principal must have the iam.serviceAccounts.getAccessToken permission or
  the role roles/iam.serviceAccounts.serviceAccountTokenCreator.

  For example:

      ./alloydb-auth-proxy \
          --impersonate-service-account=impersonated@my-project.iam.gserviceaccount.com
          projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE

  In addition, the flag supports an impersonation delegation chain where the
  value is a comma-separated list of service accounts. The first service
  account in the list is the impersonation target. Each subsequent service
  account is a delegate to the previous service account. When delegation is
  used, each delegate must have the permissions named above on the service
  account it is delegating to.

  For example:

      ./alloydb-auth-proxy \
          --impersonate-service-account=SERVICE_ACCOUNT_1,SERVICE_ACCOUNT_2,SERVICE_ACCOUNT_3
          projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE

  In this example, the environment's IAM principal impersonates
  SERVICE_ACCOUNT_3 which impersonates SERVICE_ACCOUNT_2 which then
  impersonates the target SERVICE_ACCOUNT_1.

Configuration using environment variables

  Instead of using CLI flags, the proxy may be configured using environment
  variables. Each environment variable uses "ALLOYDB_PROXY" as a prefix and
  is the uppercase version of the flag using underscores as word delimiters.
  For example, the --structured-logs flag may be set with the environment
  variable ALLOYDB_PROXY_STRUCTURED_LOGS. An invocation of the proxy using
  environment variables would look like the following:

      ALLOYDB_PROXY_STRUCTURED_LOGS=true \
          ./alloydb-auth-proxy \
          projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE

  In addition to CLI flags, instance URIs may also be specified with
  environment variables. If invoking the proxy with only one instance URI,
  use ALLOYDB_PROXY_INSTANCE_URI. For example:

      ALLOYDB_PROXY_INSTANCE_URI=projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE \
          ./alloydb-auth-proxy

  If multiple instance URIs are used, add the index of the instance URI as a
  suffix. For example:

      ALLOYDB_PROXY_INSTANCE_URI_0=projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE1 \
      ALLOYDB_PROXY_INSTANCE_URI_1=projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE2 \
          ./alloydb-auth-proxy

Configuration using a configuration file

  Instead of using CLI flags, the Proxy may be configured using a configuration
  file. The configuration file is a TOML, YAML or JSON file with the same keys
  as the environment variables. The configuration file is specified with the
  --config-file flag. An invocation of the Proxy using a configuration file
  would look like the following:

      ./alloydb-auth-proxy --config-file=config.toml

  The configuration file may look like the following:

      instance-uri = "<INSTANCE_URI>"
      auto-iam-authn = true

  If multiple instance URIs are used, add the index of the instance URI as a
  suffix. For example:

      instance-uri-0 = "<INSTANCe_URI_1>"
      instance-uri-1 = "<INSTANCE_URI_2>"

  The configuration file may also contain the same keys as the environment
  variables and flags. For example:

      auto-iam-authn = true
      debug = true
      max-connections = 5

Localhost Admin Server

  The Proxy includes support for an admin server on localhost. By default,
  the admin server is not enabled. To enable the server, pass the --debug or
  --quitquitquit flag. This will start the server on localhost at port 9091.
  To change the port, use the --admin-port flag.

  When --debug is set, the admin server enables Go's profiler available at
  /debug/pprof/.

  See the documentation on pprof for details on how to use the
  profiler at https://pkg.go.dev/net/http/pprof.

  When --quitquitquit is set, the admin server adds an endpoint at
  /quitquitquit. The admin server exits gracefully when it receives a POST
  request at /quitquitquit.

Debug logging

  On occasion, it can help to enable debug logging which will report on
  internal certificate refresh operations. To enable debug logging, use:

      ./alloydb-auth-proxy <INSTANCE_URI> --debug-logs


Waiting for Startup

  See the wait subcommand's help for details.

(*) indicates a flag that may be used as a query parameter

Third Party Licenses

  To view all licenses for third party dependencies used within this
  distribution please see:

  https://storage.googleapis.com/alloydb-auth-proxy/v1.13.10/third_party/licenses.tar.gz 

Static Connection Info

  In development contexts, it can be helpful to populate the Proxy with static
  connection info. This is a *dev-only* feature and NOT for use in production.
  The file format is subject to breaking changes.

  The format is:

  {
    "publicKey": "<PEM Encoded public RSA key>",
    "privateKey": "<PEM Encoded private RSA key>",
    "projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>": {
        "ipAddress": "<PSA-based private IP address>",
        "publicIpAddress": "<public IP address>",
        "pscInstanceConfig": {
            "pscDnsName": "<PSC DNS name>"
        },
        "pemCertificateChain": [
            "<client cert>", "<intermediate cert>", "<CA cert>"
        ],
        "caCert": "<CA cert>"
    }
  }


```
alloydb-auth-proxy instance_uri... [flags]
```

### Options

```
  -a, --address string                       (*) Address on which to bind AlloyDB instance listeners. (default "127.0.0.1")
      --admin-port string                    Port for localhost-only admin server (default "9091")
      --alloydbadmin-api-endpoint string     When set, the proxy uses this host as the base API path. (default "https://alloydb.googleapis.com")
  -i, --auto-iam-authn                       (*) Enables Automatic IAM Authentication for all instances
      --config-file string                   Path to a TOML file containing configuration options.
  -c, --credentials-file string              Path to a service account key to use for authentication.
      --debug                                Enable pprof on the localhost admin server
      --debug-logs                           Enable debug logging
      --disable-built-in-telemetry           Disables the internal metric reporter
      --disable-metrics                      Disable Cloud Monitoring integration (used with telemetry-project)
      --disable-traces                       Disable Cloud Trace integration (used with telemetry-project)
      --exit-zero-sigterm                    Exit with 0 exit code when Sigterm received (default is 143)
      --fuse string                          Mount a directory at the path using FUSE to access AlloyDB instances.
      --fuse-tmp-dir string                  Temp dir for Unix sockets created with FUSE (default "/tmp/alloydb-tmp")
  -g, --gcloud-auth                          Use gcloud's user credentials as a source of IAM credentials.
                                             NOTE: this flag is a legacy feature and generally should not be used.
                                             Instead prefer Application Default Credentials
                                             (enabled with: gcloud auth application-default login) which
                                             the Proxy will then pick-up automatically.
      --health-check                         Enables HTTP endpoints /startup, /liveness, and /readiness
                                             that report on the proxy's health. Endpoints are available on localhost
                                             only. Uses the port specified by the http-port flag.
  -h, --help                                 Display help information for alloydb-auth-proxy
      --http-address string                  Address for Prometheus and health check server (default "localhost")
      --http-port string                     Port for the Prometheus server to use (default "9090")
      --impersonate-service-account string   Comma separated list of service accounts to impersonate. Last value
                                             +is the target account.
  -j, --json-credentials string              Use service account key JSON as a source of IAM credentials.
      --lazy-refresh                         Configure a lazy refresh where connection info is retrieved only if
                                             the cached copy has expired. Use this setting in environments where the
                                             CPU may be throttled and a background refresh cannot run reliably
                                             (e.g., Cloud Run)
      --max-connections uint                 Limits the number of connections by refusing any additional connections.
                                             When this flag is not set, there is no limit.
      --max-sigterm-delay duration           Maximum amount of time to wait after for any open connections
                                             to close after receiving a TERM signal. The proxy will shut
                                             down when the number of open connections reaches 0 or when
                                             the maximum time has passed. Defaults to 0s.
      --min-sigterm-delay duration           The number of seconds to accept new connections after receiving a TERM
                                             signal. Defaults to 0s.
  -p, --port int                             (*) Initial port to use for listeners. Subsequent listeners increment from this value. (default 5432)
      --prometheus                           Enable Prometheus HTTP endpoint /metrics
      --prometheus-namespace string          Use the provided Prometheus namespace for metrics
      --psc                                  (*) Connect to the PSC endpoint for all instances
      --public-ip                            (*) Connect to the public ip address for all instances
      --quiet                                Log error messages only
      --quitquitquit                         Enable quitquitquit endpoint on the localhost admin server
      --run-connection-test                  Runs a connection test
                                             against all specified instances. If an instance is unreachable, the Proxy exits with a failure
                                             status code.
      --static-connection-info string        JSON file with static connection info. See --help for format.
  -l, --structured-logs                      Enable structured logs using the LogEntry format
      --telemetry-prefix string              Prefix to use for Cloud Monitoring metrics.
      --telemetry-project string             Enable Cloud Monitoring and Cloud Trace integration with the provided project ID.
      --telemetry-sample-rate int            Configure the denominator of the probabilistic sample rate of traces sent to Cloud Trace
                                             (e.g., 10,000 traces 1/10,000 calls). (default 10000)
  -t, --token string                         Bearer token used for authorization.
  -u, --unix-socket string                   (*) Enables Unix sockets for all listeners using the provided directory.
      --user-agent string                    Space separated list of additional user agents, e.g. custom-agent/0.0.1
  -v, --version                              Print the alloydb-auth-proxy version
```

### SEE ALSO

* [alloydb-auth-proxy wait](alloydb-auth-proxy_wait.md)	 - Wait for another Proxy process to start

