package provider_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func TestSelectProxyDoesNotAcquireForOff(t *testing.T) {
	use, err := provider.SelectProxy(provider.ProxyModeOff, true)
	if err != nil || use {
		t.Fatalf("off must never acquire a proxy: use=%v err=%v", use, err)
	}
	use, err = provider.SelectProxy("", true)
	if err != nil || use {
		t.Fatalf("empty mode must behave as off: use=%v err=%v", use, err)
	}
}

func TestSelectProxyPrefersProxyAndFallsBackDirect(t *testing.T) {
	use, err := provider.SelectProxy(provider.ProxyModePreferred, true)
	if err != nil || !use {
		t.Fatalf("preferred with healthy proxy must use it: use=%v err=%v", use, err)
	}
	use, err = provider.SelectProxy(provider.ProxyModePreferred, false)
	if err != nil || use {
		t.Fatalf("preferred without healthy proxy must fall back direct: use=%v err=%v", use, err)
	}
}

func TestSelectProxyRequiredFailsClosed(t *testing.T) {
	use, err := provider.SelectProxy(provider.ProxyModeRequired, false)
	if use || !errors.Is(err, provider.ErrProxyRequired) {
		t.Fatalf("required without healthy proxy must fail closed: use=%v err=%v", use, err)
	}
	use, err = provider.SelectProxy(provider.ProxyModeRequired, true)
	if err != nil || !use {
		t.Fatalf("required with healthy proxy must use it: use=%v err=%v", use, err)
	}
}

func TestParseProxyMode(t *testing.T) {
	for _, value := range []string{"off", "preferred", "required", "REQUIRED"} {
		if _, ok := provider.ParseProxyMode(value); !ok {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"", "auto", "on"} {
		if _, ok := provider.ParseProxyMode(value); ok {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func TestBuildProxyURLValidatesEndpoint(t *testing.T) {
	raw, err := provider.BuildProxyURL("203.0.113.10", 8080, "user", "p@ss")
	if err != nil {
		t.Fatalf("build proxy URL: %v", err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}
	if parsed.Scheme != "http" || parsed.Host != "203.0.113.10:8080" || parsed.User.Username() != "user" {
		t.Fatalf("unexpected proxy URL: %s", raw)
	}
	if _, err := provider.BuildProxyURL("http://proxy.example:8080", 8080, "", ""); err == nil {
		t.Fatal("expected scheme to be rejected")
	}
	if _, err := provider.BuildProxyURL("proxy.example", 70000, "", ""); err == nil {
		t.Fatal("expected invalid port to be rejected")
	}
	if _, err := provider.BuildProxyURL("user:pass@proxy.example", 8080, "", ""); err == nil {
		t.Fatal("expected credentials in address to be rejected")
	}
}

func TestClientForProxyUsesHTTPProxy(t *testing.T) {
	raw, err := provider.BuildProxyURL("203.0.113.10", 8080, "user", "pass")
	if err != nil {
		t.Fatalf("build proxy URL: %v", err)
	}
	client := provider.ClientForProxy(raw, 5*time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.Proxy == nil {
		t.Fatal("expected an HTTP transport with proxy configured")
	}
	request, _ := http.NewRequest(http.MethodGet, "https://provider.example", nil)
	proxyURL, err := transport.Proxy(request)
	if err != nil || proxyURL.String() != raw {
		t.Fatalf("unexpected proxy: %v err=%v", proxyURL, err)
	}
}
