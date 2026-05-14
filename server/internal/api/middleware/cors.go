package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS enables credentialed CORS for the configured frontend origin and safe
// local development origins. It echoes back the request Origin when allowed.
func CORS(defaultFrontendURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" && isAllowedOrigin(origin, defaultFrontendURL) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Api-Key")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string, defaultFrontendURL string) bool {
	u, err := url.Parse(origin)
	if err != nil || u == nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	host := u.Hostname()
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}

	def, err := url.Parse(defaultFrontendURL)
	if err != nil || def == nil {
		return strings.EqualFold(origin, defaultFrontendURL)
	}
	if strings.EqualFold(origin, defaultFrontendURL) {
		return true
	}
	return strings.EqualFold(u.Scheme, def.Scheme) && strings.EqualFold(u.Hostname(), def.Hostname())
}
