package serve

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewHandler creates the HTTP handler stack for the dev server.
// It reverse-proxies asset requests to the Vite dev server and
// serves built files for everything else.
func NewHandler(outputDir string, proxyPort int) http.Handler {
	mux := http.NewServeMux()

	// Static file server for built output
	fileServer := http.FileServer(http.Dir(outputDir))

	if proxyPort > 0 {
		proxyTarget, _ := url.Parse(fmt.Sprintf("http://localhost:%d", proxyPort))
		proxy := httputil.NewSingleHostReverseProxy(proxyTarget)

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if shouldProxy(r.URL.Path) {
				proxy.ServeHTTP(w, r)
				return
			}

			// Serve built files
			fileServer.ServeHTTP(w, r)
		})
	} else {
		mux.Handle("/", fileServer)
	}

	return mux
}

// shouldProxy returns true for paths that should be forwarded to the Vite dev server.
func shouldProxy(path string) bool {
	if strings.HasPrefix(path, "/assets/") {
		return true
	}
	if strings.HasPrefix(path, "/@vite/") {
		return true
	}
	if strings.HasPrefix(path, "/@fs/") {
		return true
	}
	if strings.HasPrefix(path, "/node_modules/") {
		return true
	}
	return false
}
