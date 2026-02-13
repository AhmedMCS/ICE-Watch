package main

import (
	"net"
	"net/http"
	"os"
	"strings"
)

// getClientIP extracts the client IP from the request.
// Only trusts proxy headers when TRUST_PROXY env is set to "true".
func getClientIP(r *http.Request) string {
	if os.Getenv("TRUST_PROXY") == "true" {
		// Check X-Forwarded-For header (for proxies)
		forwarded := r.Header.Get("X-Forwarded-For")
		if forwarded != "" {
			// Take the first IP in the chain
			parts := strings.SplitN(forwarded, ",", 2)
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}

		// Check X-Real-IP header
		realIP := r.Header.Get("X-Real-IP")
		if realIP != "" {
			return realIP
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
