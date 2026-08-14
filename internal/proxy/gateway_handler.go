package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type GatewayHandler struct {
	proxy *httputil.ReverseProxy
}

func NewGatewayHandler(targetURL string) (*GatewayHandler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	return &GatewayHandler{
		proxy: proxy,
	}, nil
}

func (h *GatewayHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.proxy.ServeHTTP(w, r)
}
