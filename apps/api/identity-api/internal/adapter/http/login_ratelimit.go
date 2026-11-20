package http

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// Login attempt limit: ten requests per minute per client.
const (
	loginAttemptsPerMinute = 10
	loginRateWindow        = time.Minute
)

// loginRateLimiter returns the middleware enforcing the login attempt limit,
// answering an over-limit request with 429.
//
// The bucket is keyed on the server-observed peer address, not on forwarding
// headers, so a client cannot forge its way into a fresh budget. Behind a
// reverse proxy every client shares the proxy address and therefore one
// budget; that is the documented limitation, matching the registration limiter.
func loginRateLimiter() func(http.Handler) http.Handler {
	return httprate.LimitBy(
		loginAttemptsPerMinute,
		loginRateWindow,
		peerAddressKey,
	)
}

// peerAddressKey returns the host part of the request's peer address, so the
// port does not split one client across buckets.
func peerAddressKey(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr, nil
	}
	return host, nil
}
