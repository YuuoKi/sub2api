package service

import (
	"context"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type httpAssetInspector struct {
	client *http.Client
}

func NewHTTPAssetInspector() AssetInspector {
	return newHTTPAssetInspector(newPublicAssetHTTPClient(5 * time.Second))
}

func newPublicAssetHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dialer := &net.Dialer{Timeout: 4 * time.Second, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid asset address: %w", err)
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil || len(addresses) == 0 {
			return nil, fmt.Errorf("resolve asset host: %w", err)
		}
		for _, address := range addresses {
			if !isPublicAssetIP(address.IP) {
				return nil, fmt.Errorf("asset host resolves to a non-public address")
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many asset redirects")
			}
			return validateAssetSourceURL(request.URL)
		},
	}
	return client
}

func newHTTPAssetInspector(client *http.Client) AssetInspector {
	return &httpAssetInspector{client: client}
}

func (i *httpAssetInspector) Inspect(ctx context.Context, rawURL string) (AssetInspection, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return AssetInspection{}, fmt.Errorf("parse asset URL: %w", err)
	}
	if err := validateAssetSourceURL(parsed); err != nil {
		return AssetInspection{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, parsed.String(), nil)
	if err != nil {
		return AssetInspection{}, fmt.Errorf("build asset metadata request: %w", err)
	}
	response, err := i.client.Do(request)
	if err != nil {
		return AssetInspection{}, fmt.Errorf("inspect asset metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return AssetInspection{}, fmt.Errorf("asset metadata returned HTTP %d", response.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || strings.TrimSpace(mediaType) == "" {
		return AssetInspection{}, fmt.Errorf("asset MIME is unavailable")
	}
	size := response.ContentLength
	if size <= 0 {
		if header := strings.TrimSpace(response.Header.Get("Content-Length")); header != "" {
			size, err = strconv.ParseInt(header, 10, 64)
		}
	}
	if err != nil || size <= 0 {
		return AssetInspection{}, fmt.Errorf("asset size is unavailable")
	}
	return AssetInspection{MIME: strings.ToLower(mediaType), SizeBytes: size}, nil
}

func validateAssetSourceURL(parsed *url.URL) error {
	if parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return fmt.Errorf("asset URL must use HTTP or HTTPS")
	}
	if parsed.User != nil {
		return fmt.Errorf("asset URL user information is not allowed")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !isPublicAssetIP(ip) {
		return fmt.Errorf("asset URL cannot target a non-public address")
	}
	return nil
}

func isPublicAssetIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast()
}
