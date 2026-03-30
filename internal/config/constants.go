package config

import "time"

// Transport modes for queue/messaging infrastructure.
const (
	TransportJetstream = "jetstream"
	TransportPoll      = "poll"
)

// Executor modes for action execution.
const (
	ExecutorLocal  = "local"
	ExecutorDocker = "docker"
	ExecutorAuto   = "auto"
)

// Network modes for executor containers.
const (
	NetworkModeNone   = "none"
	NetworkModeBridge = "bridge"
	NetworkModeCustom = "custom"
)

// Browser modes.
const (
	BrowserModeAuto     = "auto"
	BrowserModeChromedp = "chromedp"
	BrowserModeHTTP     = "http"
)

// Default configuration values.
const (
	DefaultListenAddr            = ":8080"
	DefaultPollInterval          = 1500 * time.Millisecond
	DefaultFlowMinActionInterval = 750 * time.Millisecond
	DefaultBrowserTimeout        = 20 * time.Second
	DefaultSearchTimeout         = 12 * time.Second
	DefaultTenant                = "default"
	DefaultActionResultTimeout   = 30 * time.Second
	DefaultSSETickInterval       = time.Second
	DefaultSSEKeepAliveInterval  = 15 * time.Second
)
