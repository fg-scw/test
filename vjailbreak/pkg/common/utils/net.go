package utils

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http/httpproxy"
)

// default timeout 30 seconds
var defaultTimeout = 30 * time.Second

type VjbNet struct {
	Client          *http.Client
	timeout         time.Duration
	Insecure        bool
	HTTPProxy       string
	HTTPSProxy      string
	NoProxy         string
	UseProxyFromEnv bool
	proxyCfg        *httpproxy.Config
}

func (v *VjbNet) getNetTransport(tlsConfig *tls.Config) *http.Transport {
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	if v.Insecure {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}
	// Configure proxy behavior using httpproxy.Config, which correctly
	// handles HTTP(S)_PROXY and NO_PROXY semantics including lists and
	// domain / CIDR matching.
	if v.UseProxyFromEnv || v.HTTPProxy != "" || v.HTTPSProxy != "" || v.NoProxy != "" {
		if v.proxyCfg == nil {
			v.proxyCfg = httpproxy.FromEnvironment()
		}
		if v.HTTPProxy != "" {
			v.proxyCfg.HTTPProxy = v.HTTPProxy
		}
		if v.HTTPSProxy != "" {
			v.proxyCfg.HTTPSProxy = v.HTTPSProxy
		}
		if v.NoProxy != "" {
			v.proxyCfg.NoProxy = v.NoProxy
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			proxyURL, err := v.proxyCfg.ProxyFunc()(req.URL)
			if err != nil {
				return nil, err
			}
			// Preserve existing logging behavior.
			if proxyURL != nil {
				fmt.Printf("Proxy config: HTTPProxy=%s, HTTPSProxy=%s, NoProxy=%s\n",
					v.proxyCfg.HTTPProxy, v.proxyCfg.HTTPSProxy, v.proxyCfg.NoProxy)
				fmt.Printf("VjbNet proxy config: HTTPProxy=%s, HTTPSProxy=%s, NoProxy=%s\n",
					v.HTTPProxy, v.HTTPSProxy, v.NoProxy)
			}
			return proxyURL, nil
		}
	}
	return transport
}

// CreateHTTPClient creates an HTTP client with TLS configuration and retry logic
func (v *VjbNet) CreateHTTPClient() error {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if v.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}
	transport := v.getNetTransport(tlsConfig)

	if v.UseProxyFromEnv {
		transport.Proxy = http.ProxyFromEnvironment
	}
	v.Client = &http.Client{
		Transport: transport,
		Timeout:   v.timeout,
	}
	return nil
}

// CreateSecureHTTPClient creates a secure HTTP client with TLS configuration
func (v *VjbNet) CreateSecureHTTPClient() error {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if v.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}

	transport := v.getNetTransport(tlsConfig)

	if v.UseProxyFromEnv {
		transport.Proxy = http.ProxyFromEnvironment
	}

	v.Client = &http.Client{
		Transport: transport,
		Timeout:   v.timeout,
	}
	return nil
}

func (v *VjbNet) SetTimeout(timeout time.Duration) {
	v.timeout = timeout
}

func (v *VjbNet) SetInsecure(insecure bool) {
	v.Insecure = insecure
}

func (v *VjbNet) SetHTTPProxy(proxy string) {
	v.HTTPProxy = proxy
}

func (v *VjbNet) SetHTTPSProxy(proxy string) {
	v.HTTPSProxy = proxy
}

func (v *VjbNet) SetNoProxy(noProxy string) {
	v.NoProxy = noProxy
}

func (v *VjbNet) SetUseProxyFromEnv(use bool) {
	v.UseProxyFromEnv = use
}

func (v *VjbNet) GetClient() *http.Client {
	return v.Client
}

func (v *VjbNet) proxy4URL(reqURL *url.URL) (*url.URL, error) {
	if v.proxyCfg == nil {
		v.proxyCfg = httpproxy.FromEnvironment()
	}

	//override the values with the ones set by the user
	//else use the ones in the ENV by default.
	if v.HTTPProxy != "" {
		v.proxyCfg.HTTPProxy = v.HTTPProxy
	}
	if v.HTTPSProxy != "" {
		v.proxyCfg.HTTPSProxy = v.HTTPSProxy
	}
	if v.NoProxy != "" {
		v.proxyCfg.NoProxy = v.NoProxy
	}
	// Delegate proxy decision to httpproxy's ProxyFunc for correct
	// NO_PROXY and scheme handling.
	proxyURL, err := v.proxyCfg.ProxyFunc()(reqURL)
	if err != nil {
		return nil, err
	}
	// Preserve historical behavior: for HTTPS requests, ensure the proxy URL
	// uses the https scheme, even though httpproxy.ProxyFunc typically returns
	// an http scheme (CONNECT over HTTP).
	if proxyURL != nil && reqURL.Scheme == "https" && proxyURL.Scheme == "http" {
		proxyURL.Scheme = "https"
	}
	if proxyURL != nil {
		fmt.Printf("Proxy config: HTTPProxy=%s, HTTPSProxy=%s, NoProxy=%s\n",
			v.proxyCfg.HTTPProxy, v.proxyCfg.HTTPSProxy, v.proxyCfg.NoProxy)
		fmt.Printf("VjbNet proxy config: HTTPProxy=%s, HTTPSProxy=%s, NoProxy=%s\n",
			v.HTTPProxy, v.HTTPSProxy, v.NoProxy)
	}
	return proxyURL, nil

}

func (v *VjbNet) IsProxyEnabled() bool {
	return v.HTTPProxy != "" || v.HTTPSProxy != ""
}

func (v *VjbNet) GetTimeout() time.Duration {
	return v.timeout
}

func NewVjbNet() *VjbNet {
	return &VjbNet{
		timeout:         defaultTimeout,
		Client:          &http.Client{},
		Insecure:        true,
		HTTPProxy:       "",
		HTTPSProxy:      "",
		NoProxy:         "",
		UseProxyFromEnv: true,
		proxyCfg:        httpproxy.FromEnvironment(),
	}
}

// NormalizeVCenterURL normalizes a vCenter host URL
func NormalizeVCenterURL(host string) (*url.URL, error) {
	rawHost := strings.TrimSpace(host)

	if !strings.Contains(rawHost, "://") {
		rawHost = "https://" + rawHost
	}

	u, err := url.Parse(rawHost)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	return u.JoinPath("sdk"), nil
}
