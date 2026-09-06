package ingest

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FetchOptions controls server-side URL fetching with SSRF guards.
type FetchOptions struct {
	Timeout      time.Duration
	MaxBytes     int64
	MaxRedirects int
	// AllowPrivate permits loopback/RFC1918/link-local targets.
	// Production must leave this false; tests use true with httptest servers.
	AllowPrivate bool
	UserAgent    string
}

func DefaultFetchOptions() FetchOptions {
	return FetchOptions{
		Timeout:      10 * time.Second,
		MaxBytes:     5 * 1024 * 1024,
		MaxRedirects: 3,
		UserAgent:    "Chimera-Ingest/1.0",
	}
}

// FetchedURL is a downloaded URL ready for extraction.
type FetchedURL struct {
	URL         string
	FinalURL    string
	ContentType string
	Body        []byte
}

// FetchURL downloads rawURL with SSRF guards: http/https only, hostname
// resolution checked against private ranges, dial-time IP re-checked
// (DNS-rebinding safe), redirect cap with re-validation, size cap.
func FetchURL(ctx context.Context, rawURL string, opts FetchOptions) (*FetchedURL, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 5 * 1024 * 1024
	}
	if opts.MaxRedirects <= 0 {
		opts.MaxRedirects = 3
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	if err := validateFetchURL(ctx, rawURL, opts.AllowPrivate); err != nil {
		return nil, err
	}
	allowPrivate := opts.AllowPrivate
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
		DisableKeepAlives: true,
		DialContext: func(dctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}
			if ip := net.ParseIP(host); ip != nil {
				if !allowPrivate && isBlockedIP(ip) {
					return nil, &ExtractionError{Filename: rawURL, Reason: "url resolves to blocked address"}
				}
			} else {
				// Resolve again at dial time (rebinding-safe) and check.
				ips, err := net.DefaultResolver.LookupIPAddr(dctx, host)
				if err != nil || len(ips) == 0 {
					return nil, &ExtractionError{Filename: rawURL, Reason: "url host does not resolve"}
				}
				for _, ip := range ips {
					if !allowPrivate && isBlockedIP(ip.IP) {
						return nil, &ExtractionError{Filename: rawURL, Reason: "url resolves to blocked address"}
					}
				}
			}
			return dialer.DialContext(dctx, network, addr)
		},
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= opts.MaxRedirects {
				return &ExtractionError{Filename: rawURL, Reason: fmt.Sprintf("too many redirects (max %d)", opts.MaxRedirects)}
			}
			if err := validateFetchURL(req.Context(), req.URL.String(), allowPrivate); err != nil {
				return err
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, &ExtractionError{Filename: rawURL, Reason: "invalid url", Err: err}
	}
	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,text/markdown,text/csv;q=0.9,*/*;q=0.1")
	resp, err := client.Do(req)
	if err != nil {
		if ee, ok := err.(*ExtractionError); ok {
			return nil, ee
		}
		return nil, &ExtractionError{Filename: rawURL, Reason: "fetch failed", Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &ExtractionError{Filename: rawURL, Reason: fmt.Sprintf("fetch returned status %d", resp.StatusCode)}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, opts.MaxBytes+1))
	if err != nil {
		return nil, &ExtractionError{Filename: rawURL, Reason: "read failed", Err: err}
	}
	if int64(len(body)) > opts.MaxBytes {
		return nil, &ExtractionError{Filename: rawURL, Reason: fmt.Sprintf("url content exceeds limit (%d bytes)", opts.MaxBytes)}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, &ExtractionError{Filename: rawURL, Reason: "empty after cleaning"}
	}
	ct := resp.Header.Get("Content-Type")
	return &FetchedURL{URL: rawURL, FinalURL: resp.Request.URL.String(), ContentType: ct, Body: body}, nil
}

func validateFetchURL(ctx context.Context, rawURL string, allowPrivate bool) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return &ExtractionError{Filename: rawURL, Reason: "invalid url"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return &ExtractionError{Filename: rawURL, Reason: fmt.Sprintf("url scheme %q not allowed (http/https only)", u.Scheme)}
	}
	if u.User != nil {
		return &ExtractionError{Filename: rawURL, Reason: "url with credentials not allowed"}
	}
	host := u.Hostname()
	if host == "" {
		return &ExtractionError{Filename: rawURL, Reason: "invalid url"}
	}
	if ip := net.ParseIP(host); ip != nil {
		if !allowPrivate && isBlockedIP(ip) {
			return &ExtractionError{Filename: rawURL, Reason: "url resolves to blocked address"}
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return &ExtractionError{Filename: rawURL, Reason: "url host does not resolve"}
	}
	if !allowPrivate {
		for _, ip := range ips {
			if isBlockedIP(ip.IP) {
				return &ExtractionError{Filename: rawURL, Reason: "url resolves to blocked address"}
			}
		}
	}
	return nil
}

var blockedCIDRs []net.IPNet

func init() {
	for _, c := range []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"127.0.0.0/8", "169.254.0.0/16", "100.64.0.0/10",
		"192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24",
		"224.0.0.0/4", "0.0.0.0/8",
		"::1/128", "fe80::/10", "fc00::/7", "ff00::/8",
	} {
		_, n, err := net.ParseCIDR(c)
		if err == nil {
			blockedCIDRs = append(blockedCIDRs, *n)
		}
	}
}

// isBlockedIP reports loopback, private, link-local, multicast, unspecified,
// CGNAT, and documentation ranges.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range blockedCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
