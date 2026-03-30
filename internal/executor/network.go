package executor

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	networkModeNone   = "none"
	networkModeBridge = "bridge"
	networkModeCustom = "custom"
)

type dockerNetworkConfig struct {
	Mode         string
	Name         string
	EnableNetRaw bool
}

func normalizeNetworkMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", networkModeNone:
		return networkModeNone
	case networkModeBridge:
		return networkModeBridge
	case networkModeCustom:
		return networkModeCustom
	default:
		return ""
	}
}

func parseBoolInput(input map[string]string, key string) (bool, error) {
	if len(input) == 0 {
		return false, nil
	}
	raw := strings.TrimSpace(input[key])
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid boolean for %s: %q", key, raw)
	}
	return value, nil
}

func (m *DockerManager) resolveNetworkConfig(input map[string]string) (dockerNetworkConfig, error) {
	mode := normalizeNetworkMode(m.networkMode)
	if mode == "" {
		return dockerNetworkConfig{}, fmt.Errorf("unsupported executor network mode %q", m.networkMode)
	}
	if mode == networkModeCustom && strings.TrimSpace(m.networkName) == "" {
		return dockerNetworkConfig{}, fmt.Errorf("executor custom network mode requires a network name")
	}

	requiresRawSocket := false
	for _, key := range []string{"requires_raw_socket", "enable_net_raw"} {
		value, err := parseBoolInput(input, key)
		if err != nil {
			return dockerNetworkConfig{}, err
		}
		if value {
			requiresRawSocket = true
		}
	}

	if requiresRawSocket && mode == networkModeNone {
		return dockerNetworkConfig{}, fmt.Errorf("raw socket tools require networked docker execution")
	}
	if requiresRawSocket && !m.enableNetRaw {
		return dockerNetworkConfig{}, fmt.Errorf("raw socket tools are disabled by executor policy")
	}

	return dockerNetworkConfig{
		Mode:         mode,
		Name:         strings.TrimSpace(m.networkName),
		EnableNetRaw: requiresRawSocket,
	}, nil
}
