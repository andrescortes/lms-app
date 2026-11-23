package transport

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewMSReverseProxy(targetURL string) (http.Handler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("parsing lms-core target URL %q: %w", targetURL, err)
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
	}
	return proxy, nil
}
