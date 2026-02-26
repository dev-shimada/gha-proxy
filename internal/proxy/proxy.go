package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	reverseProxy *httputil.ReverseProxy
	debug        bool
}

func New(backendURL string, debug bool) (*Proxy, error) {
	target, err := url.Parse(backendURL)
	if err != nil {
		return nil, err
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("proxy error", "error", err, "path", r.URL.Path)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return &Proxy{
		reverseProxy: rp,
		debug:        debug,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Header.Del("Authorization")

	if p.debug {
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		slog.Debug("proxying request to backend",
			"method", r.Method,
			"path", r.URL.Path,
			"headers", headers,
		)
	}

	p.reverseProxy.ServeHTTP(w, r)
}
