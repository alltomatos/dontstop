package provider

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProxyMode controls whether a provider account uses its assigned proxy.
type ProxyMode string

const (
	ProxyModeOff       ProxyMode = "off"
	ProxyModePreferred ProxyMode = "preferred"
	ProxyModeRequired  ProxyMode = "required"
	ProxyModeManual    ProxyMode = "manual"
	ProxyModeAuto      ProxyMode = "auto"
	ProxyModeInherit   ProxyMode = "inherit"
)

var (
	// ErrProxyRequired is returned when a required proxy is unavailable.
	ErrProxyRequired = errors.New("proxy required but no healthy proxy is assigned")
	// ErrInvalidProxyMode is returned for an unknown proxy mode.
	ErrInvalidProxyMode = errors.New("invalid proxy mode")
)

// NormalizeProxyMode defaults an empty or inherited value to off. Inherit is a
// declaration resolved by ResolveEffectiveMode, never a runtime behavior.
func NormalizeProxyMode(value string) ProxyMode {
	switch ProxyMode(strings.ToLower(strings.TrimSpace(value))) {
	case ProxyModePreferred:
		return ProxyModePreferred
	case ProxyModeRequired:
		return ProxyModeRequired
	case ProxyModeManual:
		return ProxyModeManual
	case ProxyModeAuto:
		return ProxyModeAuto
	default:
		return ProxyModeOff
	}
}

// ParseProxyMode validates an explicit proxy mode, including inherit.
func ParseProxyMode(value string) (ProxyMode, bool) {
	switch ProxyMode(strings.ToLower(strings.TrimSpace(value))) {
	case ProxyModeOff:
		return ProxyModeOff, true
	case ProxyModePreferred:
		return ProxyModePreferred, true
	case ProxyModeRequired:
		return ProxyModeRequired, true
	case ProxyModeManual:
		return ProxyModeManual, true
	case ProxyModeAuto:
		return ProxyModeAuto, true
	case ProxyModeInherit:
		return ProxyModeInherit, true
	default:
		return "", false
	}
}

// ResolveEffectiveMode expands inherit into the provider default (ADR-024).
func ResolveEffectiveMode(accountMode string, providerAuto bool) ProxyMode {
	mode, ok := ParseProxyMode(accountMode)
	if !ok {
		return ProxyModeOff
	}
	if mode == ProxyModeInherit {
		if providerAuto {
			return ProxyModeAuto
		}
		return ProxyModeOff
	}
	return mode
}

// RequiresProxy reports whether the mode fails closed without a healthy proxy.
func RequiresProxy(mode ProxyMode) bool {
	switch mode {
	case ProxyModeRequired, ProxyModeManual, ProxyModeAuto:
		return true
	default:
		return false
	}
}

// SelectProxy decides whether the account must use its assigned proxy.
// It never acquires a proxy for off, falls back to direct for preferred, and
// fails closed for required/manual/auto.
func SelectProxy(mode ProxyMode, healthy bool) (bool, error) {
	switch mode {
	case "", ProxyModeOff, ProxyModeInherit:
		return false, nil
	case ProxyModePreferred:
		return healthy, nil
	case ProxyModeRequired, ProxyModeManual, ProxyModeAuto:
		if !healthy {
			return false, ErrProxyRequired
		}
		return true, nil
	default:
		return false, fmt.Errorf("%w: %q", ErrInvalidProxyMode, mode)
	}
}

// BuildProxyURL validates the endpoint and builds an HTTP proxy URL. It never
// includes the proxy credentials in errors.
func BuildProxyURL(address string, port int, username, password string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", fmt.Errorf("proxy address is required")
	}
	if strings.ContainsAny(address, "/:@ \t\r\n") {
		return "", fmt.Errorf("proxy address must be a host or IP without scheme, port or credentials")
	}
	if net.ParseIP(address) == nil && !isHostname(address) {
		return "", fmt.Errorf("proxy address is not a valid host or IP")
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("proxy port must be between 1 and 65535")
	}
	proxyURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(address, strconv.Itoa(port)),
	}
	if username != "" || password != "" {
		proxyURL.User = url.UserPassword(username, password)
	}
	return proxyURL.String(), nil
}

func isHostname(value string) bool {
	if len(value) > 253 {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			isAlpha := (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
			isDigit := char >= '0' && char <= '9'
			if !isAlpha && !isDigit && char != '-' {
				return false
			}
		}
	}
	return true
}

var proxyClients sync.Map

// ClientForProxy returns a cached HTTP client routed through proxyURL.
func ClientForProxy(proxyURL string, timeout time.Duration) *http.Client {
	if proxyURL == "" {
		return &http.Client{Timeout: timeout}
	}
	if cached, ok := proxyClients.Load(proxyURL); ok {
		return cached.(*http.Client)
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return &http.Client{Timeout: timeout}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(parsed)
	client := &http.Client{Timeout: timeout, Transport: transport}
	actual, _ := proxyClients.LoadOrStore(proxyURL, client)
	return actual.(*http.Client)
}

func clientFor(base *http.Client, account Account) *http.Client {
	if account.ProxyURL == "" {
		return base
	}
	timeout := 60 * time.Second
	if base != nil && base.Timeout > 0 {
		timeout = base.Timeout
	}
	return ClientForProxy(account.ProxyURL, timeout)
}
