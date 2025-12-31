package gateway

import (
	"time"

	"github.com/julienstroheker/AzHexGate/internal/config"
	"github.com/julienstroheker/AzHexGate/internal/httpclient"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	"github.com/julienstroheker/AzHexGate/internal/relay"
)

// Client provides methods to interact with the Gateway API
type Client struct {
	baseURL    string
	httpClient *httpclient.Client
	mode       config.Mode
	listener   relay.Listener
}

// Options contains configuration for the Gateway API client
type Options struct {
	// BaseURL is the base URL of the Gateway API (optional, defaults to http://localhost:8080)
	BaseURL string

	// Timeout is the HTTP request timeout (optional, defaults to 30s)
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts (optional, defaults to 3)
	MaxRetries int

	// Logger is used for debug logging (optional)
	Logger *logging.Logger

	// Mode is the operational mode (optional, defaults to remote)
	Mode config.Mode
}

// NewClient creates a new Gateway API client
func NewClient(opts *Options) *Client {
	if opts == nil {
		opts = &Options{}
	}

	// Set defaults
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxRetries := opts.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	mode := opts.Mode
	if mode == config.Mode("") {
		mode = config.ModeRemote
	}

	// Create HTTP client with policies
	httpOpts := &httpclient.Options{
		Timeout:    timeout,
		MaxRetries: maxRetries,
		RetryDelay: time.Second,
		Logger:     opts.Logger,
		UserAgent:  "azhexgate-client/1.0",
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpclient.NewClient(httpOpts),
		mode:       mode,
	}
}
