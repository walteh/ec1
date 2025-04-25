package config

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
)

var (
	defaultAllowFrom                   = "127.0.0.1/32"         // allowed IPs to connect to the proxy
	defaultAllowHealthcheck            = false                  // allow health check requests (HEAD http://localhost:55555/health)
	defaultLogJSON                     = false                  // if true, log in JSON format
	defaultLogLevel                    = "INFO"                 // log level as string
	defaultListenIP                    = "127.0.0.1"            // ip address to bind the server to
	defaultProxyPort                   = uint(2375)             // tcp port to listen on
	defaultSocketPath                  = "/var/run/docker.sock" // path to the unix socket
	defaultShutdownGraceTime           = uint(10)               // Maximum time in seconds to wait for the server to shut down gracefully
	defaultWatchdogInterval            = uint(0)                // watchdog interval in seconds (0 to disable)
	defaultStopOnWatchdog              = false                  // set to true to stop the program when the socket gets unavailable (otherwise log only)
	defaultProxySocketEndpoint         = ""                     // empty string means no socket listener, but regular TCP listener
	defaultProxySocketEndpointFileMode = uint(0o400)            // set the file mode of the unix socket endpoint
)

type Config struct {
	AllowedRequests             map[string]*regexp.Regexp
	AllowFrom                   string
	AllowHealthcheck            bool
	StopOnWatchdog              bool
	ShutdownGraceTime           uint
	WatchdogInterval            uint
	ListenIP                    string
	ProxyPort                   uint
	SocketPath                  string
	ProxySocketEndpoint         string
	ProxySocketEndpointFileMode os.FileMode
	AllowedSSHChannelTypes      []string
	AllowedSSHUsers             map[string][]string
}

// used for list of allowed requests
type methodRegex struct {
	method               string
	regexStringFromEnv   string
	regexStringFromParam string
}

// mr is the allowlist of requests per http method
// default: regexStringFromEnv and regexStringFromParam are empty, so regexCompiled stays nil and the request is blocked
// if regexStringParam is set with a command line parameter, all requests matching the method and path matching the regex are allowed
// else if regexStringEnv from Environment ist checked
var mr = []methodRegex{
	{method: http.MethodGet},
	{method: http.MethodHead},
	{method: http.MethodPost},
	{method: http.MethodPut},
	{method: http.MethodPatch},
	{method: http.MethodDelete},
	{method: http.MethodConnect},
	{method: http.MethodTrace},
	{method: http.MethodOptions},
}

func NewDefaultConfig() *Config {
	return &Config{
		AllowedRequests:             make(map[string]*regexp.Regexp),
		AllowFrom:                   defaultAllowFrom,
		AllowHealthcheck:            defaultAllowHealthcheck,
		ListenIP:                    defaultListenIP,
		ProxyPort:                   defaultProxyPort,
		SocketPath:                  defaultSocketPath,
		ProxySocketEndpoint:         defaultProxySocketEndpoint,
		ProxySocketEndpointFileMode: os.FileMode(defaultProxySocketEndpointFileMode),
		ShutdownGraceTime:           defaultShutdownGraceTime,
		WatchdogInterval:            defaultWatchdogInterval,
		StopOnWatchdog:              defaultStopOnWatchdog,
	}
}

func (cfg *Config) ListenAddress() string {
	return fmt.Sprintf("%s:%d", cfg.ListenIP, cfg.ProxyPort)
}

func (cfg *Config) Validate() error {

	if net.ParseIP(cfg.ListenIP) == nil {
		return fmt.Errorf("invalid IP \"%s\" for listenip", cfg.ListenIP)
	}
	if cfg.ProxyPort < 1 || cfg.ProxyPort > 65535 {
		return errors.New("port number has to be between 1 and 65535")
	}

	if cfg.ProxySocketEndpointFileMode > 0o777 {
		return errors.New("file mode has to be between 0 and 0o777")
	}

	// compile regexes for allowed requests
	cfg.AllowedRequests = make(map[string]*regexp.Regexp)
	for _, rx := range mr {
		if rx.regexStringFromParam != "" {
			r, err := regexp.Compile("^" + rx.regexStringFromParam + "$")
			if err != nil {
				return fmt.Errorf("invalid regex \"%s\" for method %s in command line parameter: %w", rx.regexStringFromParam, rx.method, err)
			}
			cfg.AllowedRequests[rx.method] = r
		} else if rx.regexStringFromEnv != "" {
			r, err := regexp.Compile("^" + rx.regexStringFromEnv + "$")
			if err != nil {
				return fmt.Errorf("invalid regex \"%s\" for method %s in env variable: %w", rx.regexStringFromParam, rx.method, err)
			}
			cfg.AllowedRequests[rx.method] = r
		}
	}
	return nil
}
