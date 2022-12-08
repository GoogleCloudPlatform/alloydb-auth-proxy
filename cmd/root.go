// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/alloydb"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/healthcheck"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/log"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/proxy"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opencensus.io/trace"
)

var (
	// versionString indicates the version of this library.
	//go:embed version.txt
	versionString string
	// metadataString indiciates additional build or distribution metadata.
	metadataString string
	userAgent      string
)

func init() {
	versionString = semanticVersion()
	userAgent = "alloy-db-auth-proxy/" + versionString
}

// semanticVersion returns the version of the proxy including an compile-time
// metadata.
func semanticVersion() string {
	v := strings.TrimSpace(versionString)
	if metadataString != "" {
		v += "+" + metadataString
	}
	return v
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := NewCommand().Execute(); err != nil {
		exit := 1
		if terr, ok := err.(*exitError); ok {
			exit = terr.Code
		}
		os.Exit(exit)
	}
}

// Command represents an invocation of the AlloyDB Auth Proxy.
type Command struct {
	*cobra.Command
	conf   *proxy.Config
	logger alloydb.Logger
	dialer alloydb.Dialer

	cleanup                    func() error
	disableTraces              bool
	telemetryTracingSampleRate int
	disableMetrics             bool
	telemetryProject           string
	telemetryPrefix            string
	prometheus                 bool
	prometheusNamespace        string
	healthCheck                bool
	httpAddress                string
	httpPort                   string

	// impersonationChain is a comma separated list of one or more service
	// accounts. The last entry in the chain is the impersonation target. Any
	// additional service accounts before the target are delegates. The
	// roles/iam.serviceAccountTokenCreator must be configured for each account
	// that will be impersonated.
	impersonationChain string
}

// Option is a function that configures a Command.
type Option func(*Command)

// WithLogger overrides the default logger.
func WithLogger(l alloydb.Logger) Option {
	return func(c *Command) {
		c.logger = l
	}
}

// WithDialer configures the Command to use the provided dialer to connect to
// AlloyDB instances.
func WithDialer(d alloydb.Dialer) Option {
	return func(c *Command) {
		c.dialer = d
	}
}

var longHelp = `
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

Health checks

    When enabling the --health-check flag, the proxy will start an HTTP server
    on localhost with three endpoints:

    - /startup: Returns 200 status when the proxy has finished starting up.
    Otherwise returns 503 status.

    - /readiness: Returns 200 status when the proxy has started, has available
    connections if max connections have been set with the --max-connections
    flag, and when the proxy can connect to all registered instances. Otherwise,
    returns a 503 status. Optionally supports a min-ready query param (e.g.,
    /readiness?min-ready=3) where the proxy will return a 200 status if the
    proxy can connect successfully to at least min-ready number of instances. If
    min-ready exceeds the number of registered instances, returns a 400.

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
    variable ALLOYDB_STRUCTURED_LOGS. An invocation of the proxy using
    environment variables would look like the following:

        ALLOYDB_STRUCTURED_LOGS=true \
            ./alloydb-auth-proxy \
            projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE

    In addition to CLI flags, instance URIs may also be specified with
    environment variables. If invoking the proxy with only one instance URI,
    use ALLOYDB_INSTANCE_URI. For example:

        ALLOYDB_INSTANCE_URI=projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE \
            ./alloydb-auth-proxy

    If multiple instance URIs are used, add the index of the instance URI as a
    suffix. For example:

        ALLOYDB_INSTANCE_URI_0=projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE1 \
        ALLOYDB_INSTANCE_URI_1=projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE2 \
            ./alloydb-auth-proxy

`

const envPrefix = "ALLOYDB"

func instanceFromEnv(args []string) []string {
	// This supports naming the first instance first with:
	//     INSTANCE_URI
	// or if that's not defined, with:
	//     INSTANCE_URI_0
	inst := os.Getenv(fmt.Sprintf("%s_INSTANCE_URI", envPrefix))
	if inst == "" {
		inst = os.Getenv(fmt.Sprintf("%s_INSTANCE_URI_0", envPrefix))
		if inst == "" {
			return nil
		}
	}
	args = append(args, inst)

	i := 1
	for {
		instN := os.Getenv(fmt.Sprintf("%s_INSTANCE_URI_%d", envPrefix, i))
		// if the next instance connection name is not defined, stop checking
		// environment variables.
		if instN == "" {
			break
		}
		args = append(args, instN)
		i++
	}
	return args
}

// NewCommand returns a Command object representing an invocation of the proxy.
func NewCommand(opts ...Option) *Command {
	cmd := &cobra.Command{
		Use:     "alloydb-auth-proxy instance_uri...",
		Version: versionString,
		Short:   "alloydb-auth-proxy provides a secure way to authorize connections to AlloyDB.",
		Long:    longHelp,
	}

	logger := log.NewStdLogger(os.Stdout, os.Stderr)
	c := &Command{
		Command: cmd,
		logger:  logger,
		cleanup: func() error { return nil },
		conf: &proxy.Config{
			UserAgent: userAgent,
		},
	}
	for _, o := range opts {
		o(c)
	}

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		// If args is not already populated, try to read from the environment.
		if len(args) == 0 {
			args = instanceFromEnv(args)
		}
		// Handle logger separately from config
		if c.conf.StructuredLogs {
			c.logger, c.cleanup = log.NewStructuredLogger()
		}
		err := parseConfig(c, c.conf, args)
		if err != nil {
			return err
		}
		// The arguments are parsed. Usage is no longer needed.
		cmd.SilenceUsage = true
		// Errors will be handled by logging from here on.
		cmd.SilenceErrors = true
		return nil
	}

	cmd.RunE = func(*cobra.Command, []string) error { return runSignalWrapper(c) }

	pflags := cmd.PersistentFlags()

	// Global-only flags
	pflags.StringVarP(&c.conf.Token, "token", "t", "",
		"Bearer token used for authorization.")
	pflags.StringVarP(&c.conf.CredentialsFile, "credentials-file", "c", "",
		"Path to a service account key to use for authentication.")
	pflags.StringVarP(&c.conf.CredentialsJSON, "json-credentials", "j", "",
		"Use service account key JSON as a source of IAM credentials.")
	pflags.BoolVarP(&c.conf.GcloudAuth, "gcloud-auth", "g", false,
		"Use gcloud's user configuration to retrieve a token for authentication.")
	pflags.BoolVarP(&c.conf.StructuredLogs, "structured-logs", "l", false,
		"Enable structured logs using the LogEntry format")
	pflags.Uint64Var(&c.conf.MaxConnections, "max-connections", 0,
		`Limits the number of connections by refusing any additional connections.
When this flag is not set, there is no limit.`)
	pflags.DurationVar(&c.conf.WaitOnClose, "max-sigterm-delay", 0,
		`Maximum amount of time to wait after for any open connections
to close after receiving a TERM signal. The proxy will shut
down when the number of open connections reaches 0 or when
the maximum time has passed. Defaults to 0s.`)
	pflags.StringVar(&c.conf.APIEndpointURL, "alloydbadmin-api-endpoint",
		"https://alloydb.googleapis.com/v1beta",
		"When set, the proxy uses this host as the base API path.")
	pflags.StringVar(&c.conf.FUSEDir, "fuse", "",
		"Mount a directory at the path using FUSE to access AlloyDB instances.")
	pflags.StringVar(&c.conf.FUSETempDir, "fuse-tmp-dir",
		filepath.Join(os.TempDir(), "csql-tmp"),
		"Temp dir for Unix sockets created with FUSE")
	pflags.StringVar(&c.impersonationChain, "impersonate-service-account", "",
		`Comma separated list of service accounts to impersonate. Last value
+is the target account.`)

	pflags.StringVar(&c.telemetryProject, "telemetry-project", "",
		"Enable Cloud Monitoring and Cloud Trace integration with the provided project ID.")
	pflags.BoolVar(&c.disableTraces, "disable-traces", false,
		"Disable Cloud Trace integration (used with telemetry-project)")
	pflags.IntVar(&c.telemetryTracingSampleRate, "telemetry-sample-rate", 10_000,
		"Configure the denominator of the probabilistic sample rate of traces sent to Cloud Trace\n(e.g., 10,000 traces 1/10,000 calls).")
	pflags.BoolVar(&c.disableMetrics, "disable-metrics", false,
		"Disable Cloud Monitoring integration (used with telemetry-project)")
	pflags.StringVar(&c.telemetryPrefix, "telemetry-prefix", "",
		"Prefix to use for Cloud Monitoring metrics.")
	pflags.BoolVar(&c.prometheus, "prometheus", false,
		"Enable Prometheus HTTP endpoint /metrics")
	pflags.StringVar(&c.prometheusNamespace, "prometheus-namespace", "",
		"Use the provided Prometheus namespace for metrics")
	pflags.StringVar(&c.httpAddress, "http-address", "localhost",
		"Address for Prometheus and health check server")
	pflags.StringVar(&c.httpPort, "http-port", "9090",
		"Port for the Prometheus server to use")
	pflags.BoolVar(&c.healthCheck, "health-check", false,
		`Enables HTTP endpoints /startup, /liveness, and /readiness
that report on the proxy's health. Endpoints are available on localhost
only. Uses the port specified by the http-port flag.`)

	// Global and per instance flags
	pflags.StringVarP(&c.conf.Addr, "address", "a", "127.0.0.1",
		"Address on which to bind AlloyDB instance listeners.")
	pflags.IntVarP(&c.conf.Port, "port", "p", 5432,
		"Initial port to use for listeners. Subsequent listeners increment from this value.")
	pflags.StringVarP(&c.conf.UnixSocket, "unix-socket", "u", "",
		`Enables Unix sockets for all listeners using the provided directory.`)

	v := viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer("-", "_")))
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	// Ignoring the error here since its only occurence is if one of the pflags
	// is nil which is never the case here.
	_ = v.BindPFlags(pflags)

	pflags.VisitAll(func(f *pflag.Flag) {
		// Override any unset flags with Viper values to use the pflags
		// object as a single source of truth.
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			pflags.Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	return c
}

func parseConfig(cmd *Command, conf *proxy.Config, args []string) error {
	// If no instance connection names were provided AND FUSE isn't enabled,
	// error.
	if len(args) == 0 && conf.FUSEDir == "" {
		return newBadCommandError("missing instance uri (e.g., projects/$PROJECTS/locations/$LOCTION/clusters/$CLUSTER/instances/$INSTANCES)")
	}

	if conf.FUSEDir != "" {
		if err := proxy.SupportsFUSE(); err != nil {
			return newBadCommandError(
				fmt.Sprintf("--fuse is not supported: %v", err),
			)
		}
	}

	if len(args) == 0 && conf.FUSEDir == "" && conf.FUSETempDir != "" {
		return newBadCommandError("cannot specify --fuse-tmp-dir without --fuse")
	}

	userHasSet := func(f string) bool {
		return cmd.PersistentFlags().Lookup(f).Changed
	}
	if userHasSet("address") && userHasSet("unix-socket") {
		return newBadCommandError("cannot specify --unix-socket and --address together")
	}
	if userHasSet("port") && userHasSet("unix-socket") {
		return newBadCommandError("cannot specify --unix-socket and --port together")
	}
	// First, validate global config.
	if ip := net.ParseIP(conf.Addr); ip == nil {
		return newBadCommandError(fmt.Sprintf("not a valid IP address: %q", conf.Addr))
	}

	// If more than one auth method is set, error.
	if conf.Token != "" && conf.CredentialsFile != "" {
		return newBadCommandError("cannot specify --token and --credentials-file flags at the same time")
	}
	if conf.Token != "" && conf.GcloudAuth {
		return newBadCommandError("cannot specify --token and --gcloud-auth flags at the same time")
	}
	if conf.CredentialsFile != "" && conf.GcloudAuth {
		return newBadCommandError("cannot specify --credentials-file and --gcloud-auth flags at the same time")
	}
	if conf.CredentialsJSON != "" && conf.Token != "" {
		return newBadCommandError("cannot specify --json-credentials and --token flags at the same time")
	}
	if conf.CredentialsJSON != "" && conf.CredentialsFile != "" {
		return newBadCommandError("cannot specify --json-credentials and --credentials-file flags at the same time")
	}
	if conf.CredentialsJSON != "" && conf.GcloudAuth {
		return newBadCommandError("cannot specify --json-credentials and --gcloud-auth flags at the same time")
	}

	if userHasSet("alloydbadmin-api-endpoint") {
		_, err := url.Parse(conf.APIEndpointURL)
		if err != nil {
			return newBadCommandError(fmt.Sprintf(
				"provided value for --alloydbadmin-api-endpoint is not a valid url, %v",
				conf.APIEndpointURL,
			))
		}

		// Remove trailing '/' if included
		if strings.HasSuffix(conf.APIEndpointURL, "/") {
			conf.APIEndpointURL = strings.TrimSuffix(conf.APIEndpointURL, "/")
		}
		cmd.logger.Infof("Using API Endpoint %v", conf.APIEndpointURL)
	}

	if userHasSet("http-port") && !userHasSet("prometheus") && !userHasSet("health-check") {
		cmd.logger.Infof("Ignoring --http-port because --prometheus or --health-check was not set")
	}

	if !userHasSet("telemetry-project") && userHasSet("telemetry-prefix") {
		cmd.logger.Infof("Ignoring --telementry-prefix as --telemetry-project was not set")
	}
	if !userHasSet("telemetry-project") && userHasSet("disable-metrics") {
		cmd.logger.Infof("Ignoring --disable-metrics as --telemetry-project was not set")
	}
	if !userHasSet("telemetry-project") && userHasSet("disable-traces") {
		cmd.logger.Infof("Ignoring --disable-traces as --telemetry-project was not set")
	}

	if cmd.impersonationChain != "" {
		accts := strings.Split(cmd.impersonationChain, ",")
		conf.ImpersonateTarget = accts[0]
		// Assign delegates if the chain is more than one account. Delegation
		// goes from last back towards target, e.g., With sa1,sa2,sa3, sa3
		// delegates to sa2, which impersonates the target sa1.
		if l := len(accts); l > 1 {
			for i := l - 1; i > 0; i-- {
				conf.ImpersonateDelegates = append(conf.ImpersonateDelegates, accts[i])
			}
		}
	}

	var ics []proxy.InstanceConnConfig
	for _, a := range args {
		// Assume no query params initially
		ic := proxy.InstanceConnConfig{
			Name: a,
		}
		// If there are query params, update instance config.
		if res := strings.SplitN(a, "?", 2); len(res) > 1 {
			ic.Name = res[0]
			q, err := url.ParseQuery(res[1])
			if err != nil {
				return newBadCommandError(fmt.Sprintf("could not parse query: %q", res[1]))
			}

			a, aok := q["address"]
			p, pok := q["port"]
			u, uok := q["unix-socket"]

			if aok && uok {
				return newBadCommandError("cannot specify both address and unix-socket query params")
			}
			if pok && uok {
				return newBadCommandError("cannot specify both port and unix-socket query params")
			}

			if aok {
				if len(a) != 1 {
					return newBadCommandError(fmt.Sprintf("address query param should be only one value: %q", a))
				}
				if ip := net.ParseIP(a[0]); ip == nil {
					return newBadCommandError(
						fmt.Sprintf("address query param is not a valid IP address: %q",
							a[0],
						))
				}
				ic.Addr = a[0]
			}

			if pok {
				if len(p) != 1 {
					return newBadCommandError(fmt.Sprintf("port query param should be only one value: %q", a))
				}
				pp, err := strconv.Atoi(p[0])
				if err != nil {
					return newBadCommandError(
						fmt.Sprintf("port query param is not a valid integer: %q",
							p[0],
						))
				}
				ic.Port = pp
			}

			if uok {
				if len(u) != 1 {
					return newBadCommandError(fmt.Sprintf("unix query param should be only one value: %q", a))
				}
				ic.UnixSocket = u[0]

			}
		}
		ics = append(ics, ic)
	}

	conf.Instances = ics
	return nil
}

// runSignalWrapper watches for SIGTERM and SIGINT and interupts execution if necessary.
func runSignalWrapper(cmd *Command) error {
	defer cmd.cleanup()
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Configure collectors before the proxy has started to ensure we are
	// collecting metrics before *ANY* AlloyDB Admin API calls are made.
	enableMetrics := !cmd.disableMetrics
	enableTraces := !cmd.disableTraces
	if cmd.telemetryProject != "" && (enableMetrics || enableTraces) {
		sd, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID:    cmd.telemetryProject,
			MetricPrefix: cmd.telemetryPrefix,
		})
		if err != nil {
			return err
		}
		if enableMetrics {
			err = sd.StartMetricsExporter()
			if err != nil {
				return err
			}
		}
		if enableTraces {
			s := trace.ProbabilitySampler(1 / float64(cmd.telemetryTracingSampleRate))
			trace.ApplyConfig(trace.Config{DefaultSampler: s})
			trace.RegisterExporter(sd)
		}
		defer func() {
			sd.Flush()
			sd.StopMetricsExporter()
		}()
	}

	var (
		needsHTTPServer bool
		mux             = http.NewServeMux()
	)
	if cmd.prometheus {
		needsHTTPServer = true
		e, err := prometheus.NewExporter(prometheus.Options{
			Namespace: cmd.prometheusNamespace,
		})
		if err != nil {
			return err
		}
		mux.Handle("/metrics", e)
	}

	shutdownCh := make(chan error)
	// watch for sigterm / sigint signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		var s os.Signal
		select {
		case s = <-signals:
		case <-ctx.Done():
			// this should only happen when the context supplied in tests in canceled
			s = syscall.SIGINT
		}
		switch s {
		case syscall.SIGINT:
			shutdownCh <- errSigInt
		case syscall.SIGTERM:
			shutdownCh <- errSigTerm
		}
	}()

	// Start the proxy asynchronously, so we can exit early if a shutdown signal is sent
	startCh := make(chan *proxy.Client)
	go func() {
		defer close(startCh)
		p, err := proxy.NewClient(ctx, cmd.dialer, cmd.logger, cmd.conf)
		if err != nil {
			shutdownCh <- fmt.Errorf("unable to start: %v", err)
			return
		}
		startCh <- p
	}()
	// Wait for either startup to finish or a signal to interupt
	var p *proxy.Client
	select {
	case err := <-shutdownCh:
		cmd.logger.Errorf("The proxy has encountered a terminal error: %v", err)
		return err
	case p = <-startCh:
		cmd.logger.Infof("The proxy has started successfully and is ready for new connections!")
	}
	defer func() {
		if cErr := p.Close(); cErr != nil {
			cmd.logger.Errorf("error during shutdown: %v", cErr)
		}
	}()

	notify := func() {}
	if cmd.healthCheck {
		needsHTTPServer = true
		hc := healthcheck.NewCheck(p, cmd.logger)
		mux.HandleFunc("/startup", hc.HandleStartup)
		mux.HandleFunc("/readiness", hc.HandleReadiness)
		mux.HandleFunc("/liveness", hc.HandleLiveness)
		notify = hc.NotifyStarted
	}

	// Start the HTTP server if anything requiring HTTP is specified.
	if needsHTTPServer {
		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%s", cmd.httpAddress, cmd.httpPort),
			Handler: mux,
		}
		// Start the HTTP server.
		go func() {
			err := server.ListenAndServe()
			if err == http.ErrServerClosed {
				return
			}
			if err != nil {
				shutdownCh <- fmt.Errorf("failed to start HTTP server: %v", err)
			}
		}()
		// Handle shutdown of the HTTP server gracefully.
		go func() {
			select {
			case <-ctx.Done():
				// Give the HTTP server a second to shutdown cleanly.
				ctx2, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if err := server.Shutdown(ctx2); err != nil {
					cmd.logger.Errorf("failed to shutdown Prometheus HTTP server: %v\n", err)
				}
			}
		}()
	}

	go func() { shutdownCh <- p.Serve(ctx, notify) }()

	err := <-shutdownCh
	switch {
	case errors.Is(err, errSigInt):
		cmd.logger.Errorf("SIGINT signal received. Shutting down...")
	case errors.Is(err, errSigTerm):
		cmd.logger.Errorf("SIGTERM signal received. Shutting down...")
	default:
		cmd.logger.Errorf("The proxy has encountered a terminal error: %v", err)
	}
	return err
}
