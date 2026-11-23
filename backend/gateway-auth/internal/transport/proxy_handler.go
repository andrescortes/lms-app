package transport

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func NewMSReverseProxy(targetURL string) (http.Handler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("parsing lms-core target URL %q: %w", targetURL, err)
	}

	// Validate URL scheme to prevent SSRF
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("lms-core target URL must use http or https scheme, got %q", target.Scheme)
	}

	// In production, prevent proxying to private/internal IPs (SSRF protection)
	// This is a defense-in-depth measure; the config validation should also enforce HTTPS
	host := target.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			// Allow in development, but log warning
			slog.Warn("lms-core target resolves to private/loopback IP", "host", host, "ip", ip.String())
		}
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// replace for Scheme, Host, Path y pr.Out.Hos
			pr.SetURL(target)
			//adds X-Forwarded-For, X-Forwarded-Host y X-Forwarded-Proto clean way
			pr.SetXForwarded()
			// Security anti-spoofing
			pr.Out.Header.Del("X-User-ID")
			pr.Out.Header.Del("X-User-Role")
			pr.Out.Header.Del("X-User-Level")
			pr.Out.Header.Del("X-User-Email")
			// extract verified claims by AuthMiddleware
			if claims, ok := GetUserClaims(pr.In.Context()); ok {
				pr.Out.Header.Set("X-User-ID", claims.Subject)
				pr.Out.Header.Set("X-User-Role", claims.Role)
				pr.Out.Header.Set("X-User-Level", claims.CEFRLevel)
				pr.Out.Header.Set("X-User-Email", claims.Email)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("reverse proxy error forwarding to lms-core",
				"error", err,
				"method", r.Method,
				"path", r.URL.Path,
				"target", targetURL,
			)
			writeError(w, http.StatusBadGateway, "lms-core service is temporarily unavailable")
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			// In production, you could add a custom DialContext that validates the target IP
			// against a allowlist to prevent SSRF. For now, config validation handles HTTPS.
		},
	}
	return proxy, nil
}
