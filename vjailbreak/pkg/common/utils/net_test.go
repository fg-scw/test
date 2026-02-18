package utils

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestNewVjbNet_Defaults(t *testing.T) {
	n := NewVjbNet()

	if n == nil {
		t.Fatalf("NewVjbNet returned nil")
	}

	if n.timeout != defaultTimeout {
		t.Errorf("expected default timeout %v, got %v", defaultTimeout, n.timeout)
	}

	if n.Client == nil {
		t.Errorf("expected non-nil http.Client")
	}

	if !n.Insecure {
		t.Errorf("expected Insecure to be true by default")
	}

	if n.HTTPProxy != "" || n.HTTPSProxy != "" || n.NoProxy != "" {
		t.Errorf("expected proxy fields to be empty by default")
	}

	if !n.UseProxyFromEnv {
		t.Errorf("expected UseProxyFromEnv to be true by default")
	}

	if n.proxyCfg == nil {
		t.Errorf("expected proxyCfg to be initialized from environment")
	}
}

func TestVjbNet_SettersAndGetters(t *testing.T) {
	n := NewVjbNet()

	// timeout
	customTimeout := 10 * time.Second
	n.SetTimeout(customTimeout)
	if got := n.GetTimeout(); got != customTimeout {
		t.Errorf("GetTimeout = %v, want %v", got, customTimeout)
	}

	// insecure
	n.SetInsecure(false)
	if n.Insecure {
		t.Errorf("expected Insecure to be false after SetInsecure(false)")
	}

	// http proxy
	httpProxy := "http-proxy.example:8080"
	n.SetHTTPProxy(httpProxy)
	if n.HTTPProxy != httpProxy {
		t.Errorf("HTTPProxy = %q, want %q", n.HTTPProxy, httpProxy)
	}

	// https proxy
	httpsProxy := "https-proxy.example:8443"
	n.SetHTTPSProxy(httpsProxy)
	if n.HTTPSProxy != httpsProxy {
		t.Errorf("HTTPSProxy = %q, want %q", n.HTTPSProxy, httpsProxy)
	}

	// no proxy
	noProxy := "no-proxy.example"
	n.SetNoProxy(noProxy)
	if n.NoProxy != noProxy {
		t.Errorf("NoProxy = %q, want %q", n.NoProxy, noProxy)
	}

	// use proxy from env
	n.SetUseProxyFromEnv(false)
	if n.UseProxyFromEnv {
		t.Errorf("expected UseProxyFromEnv to be false after SetUseProxyFromEnv(false)")
	}
}

func TestVjbNet_IsProxyEnabled(t *testing.T) {
	n := NewVjbNet()

	if n.IsProxyEnabled() {
		t.Errorf("expected IsProxyEnabled to be false when no proxies are set")
	}

	n.SetHTTPProxy("http-proxy.example:8080")
	if !n.IsProxyEnabled() {
		t.Errorf("expected IsProxyEnabled to be true when HTTP proxy is set")
	}

	n.SetHTTPProxy("")
	n.SetHTTPSProxy("https-proxy.example:8443")
	if !n.IsProxyEnabled() {
		t.Errorf("expected IsProxyEnabled to be true when HTTPS proxy is set")
	}
}

func TestVjbNet_CreateHTTPClient(t *testing.T) {
	n := NewVjbNet()
	n.SetTimeout(5 * time.Second)
	n.SetInsecure(false)

	if err := n.CreateHTTPClient(); err != nil {
		t.Fatalf("CreateHTTPClient returned error: %v", err)
	}

	client := n.GetClient()
	if client == nil {
		t.Fatalf("expected non-nil http.Client after CreateHTTPClient")
	}

	if client.Timeout != n.timeout {
		t.Errorf("client.Timeout = %v, want %v", client.Timeout, n.timeout)
	}

	// Ensure Transport is set
	if client.Transport == nil {
		t.Errorf("expected non-nil Transport on http.Client")
	}
}

func TestVjbNet_CreateSecureHTTPClient(t *testing.T) {
	n := NewVjbNet()
	n.SetTimeout(7 * time.Second)
	n.SetInsecure(false)

	if err := n.CreateSecureHTTPClient(); err != nil {
		t.Fatalf("CreateSecureHTTPClient returned error: %v", err)
	}

	client := n.GetClient()
	if client == nil {
		t.Fatalf("expected non-nil http.Client after CreateSecureHTTPClient")
	}

	if client.Timeout != n.timeout {
		t.Errorf("client.Timeout = %v, want %v", client.Timeout, n.timeout)
	}

	// Ensure Transport is set
	if client.Transport == nil {
		t.Errorf("expected non-nil Transport on http.Client")
	}
}

func TestVjbNet_proxy4URL_HTTPAndHTTPS(t *testing.T) {
	n := NewVjbNet()

	httpProxy := "http-proxy.example:8080"
	httpsProxy := "https-proxy.example:8443"
	n.SetHTTPProxy(httpProxy)
	n.SetHTTPSProxy(httpsProxy)

	// Force proxyCfg to be initialized and overridden by setters via proxy4URL
	uHTTP, err := url.Parse("http://some-http-host")
	if err != nil {
		t.Fatalf("failed to parse HTTP URL: %v", err)
	}
	proxyHTTP, err := n.proxy4URL(uHTTP)
	if err != nil {
		t.Fatalf("proxy4URL(http) returned error: %v", err)
	}
	if proxyHTTP == nil {
		t.Fatalf("expected non-nil proxy for HTTP URL")
	}
	if proxyHTTP.Scheme != "http" || proxyHTTP.Host != httpProxy {
		t.Errorf("HTTP proxy = %s://%s, want http://%s", proxyHTTP.Scheme, proxyHTTP.Host, httpProxy)
	}

	uHTTPS, err := url.Parse("https://some-https-host")
	if err != nil {
		t.Fatalf("failed to parse HTTPS URL: %v", err)
	}
	proxyHTTPS, err := n.proxy4URL(uHTTPS)
	if err != nil {
		t.Fatalf("proxy4URL(https) returned error: %v", err)
	}
	if proxyHTTPS == nil {
		t.Fatalf("expected non-nil proxy for HTTPS URL")
	}
	if proxyHTTPS.Scheme != "https" || proxyHTTPS.Host != httpsProxy {
		t.Errorf("HTTPS proxy = %s://%s, want https://%s", proxyHTTPS.Scheme, proxyHTTPS.Host, httpsProxy)
	}
}

func TestVjbNet_proxy4URL_NoProxyConfigPropagation(t *testing.T) {
	n := NewVjbNet()

	httpProxy := "http-proxy.example:8080"
	httpsProxy := "https-proxy.example:8443"
	noProxy := "no-proxy.example"

	n.SetHTTPProxy(httpProxy)
	n.SetHTTPSProxy(httpsProxy)
	n.SetNoProxy(noProxy)

	// Call proxy4URL once to ensure proxyCfg is initialized and updated
	u, err := url.Parse("http://some-http-host")
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	if _, err := n.proxy4URL(u); err != nil {
		t.Fatalf("proxy4URL returned error: %v", err)
	}

	if n.proxyCfg == nil {
		t.Fatalf("expected proxyCfg to be initialized")
	}
	if n.proxyCfg.HTTPProxy != httpProxy {
		t.Errorf("proxyCfg.HTTPProxy = %q, want %q", n.proxyCfg.HTTPProxy, httpProxy)
	}
	if n.proxyCfg.HTTPSProxy != httpsProxy {
		t.Errorf("proxyCfg.HTTPSProxy = %q, want %q", n.proxyCfg.HTTPSProxy, httpsProxy)
	}
	if n.proxyCfg.NoProxy != noProxy {
		t.Errorf("proxyCfg.NoProxy = %q, want %q", n.proxyCfg.NoProxy, noProxy)
	}
}

func TestVjbNet_NoProxyBypassesProxy(t *testing.T) {
	n := NewVjbNet()

	httpProxy := "http-proxy.example:8080"
	noProxyHost := "no-proxy.example:80"

	// Configure HTTP proxy and NoProxy host
	n.SetHTTPProxy(httpProxy)
	n.SetNoProxy("no-proxy.example:80")

	// Disable UseProxyFromEnv so our custom Proxy function is preserved
	n.SetUseProxyFromEnv(false)

	// Warm up proxyCfg so that NoProxy is propagated into it
	warmURL, err := url.Parse("http://warmup-host")
	if err != nil {
		t.Fatalf("failed to parse warmup URL: %v", err)
	}
	if _, err := n.proxy4URL(warmURL); err != nil {
		t.Fatalf("proxy4URL warmup returned error: %v", err)
	}

	// Build client and transport with these settings
	if err := n.CreateHTTPClient(); err != nil {
		t.Fatalf("CreateHTTPClient returned error: %v", err)
	}

	client := n.GetClient()
	if client == nil {
		t.Fatalf("expected non-nil client")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport is not *http.Transport")
	}
	if transport.Proxy == nil {
		t.Fatalf("expected non-nil Proxy function when HTTPProxy and NoProxy are set")
	}

	// Request to a host that matches NoProxy should bypass proxy (return nil, nil)
	reqURL, err := url.Parse("http://" + noProxyHost)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	proxyURL, err := transport.Proxy(&http.Request{URL: reqURL})
	if err != nil {
		t.Fatalf("Proxy callback returned error: %v", err)
	}
	if proxyURL != nil {
		t.Errorf("expected nil proxy for NoProxy host, got %v", proxyURL)
	}

	// Request to a different host should use the configured proxy
	otherURL, err := url.Parse("http://other-host.example")
	if err != nil {
		t.Fatalf("failed to parse other URL: %v", err)
	}
	proxyURL, err = transport.Proxy(&http.Request{URL: otherURL})
	if err != nil {
		t.Fatalf("Proxy callback for other host returned error: %v", err)
	}
	if proxyURL == nil {
		t.Fatalf("expected non-nil proxy for non-NoProxy host")
	}
	if proxyURL.Scheme != "http" || proxyURL.Host != httpProxy {
		t.Errorf("proxy for other host = %s://%s, want http://%s", proxyURL.Scheme, proxyURL.Host, httpProxy)
	}
}

func TestVjbNet_NoProxy_CommaSeparatedAndCIDR(t *testing.T) {
	n := NewVjbNet()

	httpProxy := "http-proxy.example:8080"
	noProxy := "example.com,10.0.0.0/8"

	// Configure HTTP proxy and comma-separated NO_PROXY including domain and CIDR
	n.SetHTTPProxy(httpProxy)
	n.SetNoProxy(noProxy)
	n.SetUseProxyFromEnv(false)

	if err := n.CreateHTTPClient(); err != nil {
		t.Fatalf("CreateHTTPClient returned error: %v", err)
	}
	client := n.GetClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport is not *http.Transport")
	}
	if transport.Proxy == nil {
		t.Fatalf("expected non-nil Proxy function when HTTPProxy and NoProxy are set")
	}

	tests := []struct {
		name        string
		rawURL      string
		wantNoProxy bool
	}{
		{"no_proxy_exact_domain", "http://example.com", true},
		{"no_proxy_subdomain", "http://sub.example.com", true},
		{"no_proxy_cidr_match", "http://10.1.2.3", true},
		{"use_proxy_other_domain", "http://other.com", false},
		{"use_proxy_other_ip", "http://192.168.1.1", false},
	}

	for _, ttcase := range tests {
		t.Run(ttcase.name, func(t *testing.T) {
			u, err := url.Parse(ttcase.rawURL)
			if err != nil {
				t.Fatalf("failed to parse URL %q: %v", ttcase.rawURL, err)
			}
			proxyURL, err := transport.Proxy(&http.Request{URL: u})
			if err != nil {
				t.Fatalf("Proxy callback returned error: %v", err)
			}
			if ttcase.wantNoProxy {
				if proxyURL != nil {
					t.Fatalf("expected nil proxy for %s, got %v", ttcase.rawURL, proxyURL)
				}
			} else {
				if proxyURL == nil {
					t.Fatalf("expected non-nil proxy for %s", ttcase.rawURL)
				}
				if proxyURL.Scheme != "http" || proxyURL.Host != httpProxy {
					t.Fatalf("proxy for %s = %s://%s, want http://%s", ttcase.rawURL, proxyURL.Scheme, proxyURL.Host, httpProxy)
				}
			}
		})
	}
}

func TestVjbNet_GetClient_Default(t *testing.T) {
	n := NewVjbNet()

	client := n.GetClient()
	if client == nil {
		t.Fatalf("expected non-nil client from GetClient on new VjbNet")
	}

	if _, ok := interface{}(client).(*http.Client); !ok {
		t.Errorf("GetClient did not return *http.Client")
	}
}
