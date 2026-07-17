package service

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestHTTPAssetInspectorUsesHEADAndCanonicalizesMIME(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodHead, request.Method)
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{"Content-Type": []string{"video/mp4; charset=binary"}, "Content-Length": []string{"2048"}},
			ContentLength: 2048,
			Body:          io.NopCloser(strings.NewReader("")),
			Request:       request,
		}, nil
	})}
	inspector := newHTTPAssetInspector(client)

	inspection, err := inspector.Inspect(context.Background(), "https://assets.example.test/result.mp4")
	require.NoError(t, err)
	require.Equal(t, AssetInspection{MIME: "video/mp4", SizeBytes: 2048}, inspection)
}

func TestPublicAssetHTTPClientRejectsAnyUnsafeDNSAnswerBeforeDial(t *testing.T) {
	for _, tt := range []struct {
		name      string
		addresses []net.IPAddr
	}{
		{name: "mixed public and cgnat", addresses: []net.IPAddr{{IP: net.IP{8, 8, 8, 8}}, {IP: net.IP{100, 64, 0, 1}}}},
		{name: "benchmark", addresses: []net.IPAddr{{IP: net.IP{198, 18, 0, 1}}}},
		{name: "ipv4 mapped ipv6", addresses: []net.IPAddr{{IP: net.IP(netip.MustParseAddr("::ffff:8.8.8.8").AsSlice())}}},
		{name: "ipv6 documentation", addresses: []net.IPAddr{{IP: net.IP(netip.MustParseAddr("2001:db8::1").AsSlice())}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dialCalls := 0
			client := newPublicAssetHTTPClientWithNetwork(5*time.Second,
				func(context.Context, string) ([]net.IPAddr, error) { return tt.addresses, nil },
				func(context.Context, string, string) (net.Conn, error) {
					dialCalls++
					return nil, errors.New("unexpected dial")
				})
			transport := client.Transport.(*http.Transport)

			_, err := transport.DialContext(context.Background(), "tcp", "assets.example.test:443")
			require.Error(t, err)
			require.Zero(t, dialCalls)
		})
	}
}

func TestPublicAssetHTTPClientPinsDialToValidatedDNSAnswer(t *testing.T) {
	dialErr := errors.New("synthetic dial stop")
	dialAddress := ""
	client := newPublicAssetHTTPClientWithNetwork(5*time.Second,
		func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.IP{8, 8, 8, 8}}, {IP: net.IP{1, 1, 1, 1}}}, nil
		},
		func(_ context.Context, _ string, address string) (net.Conn, error) {
			dialAddress = address
			return nil, dialErr
		})
	transport := client.Transport.(*http.Transport)

	_, err := transport.DialContext(context.Background(), "tcp", "assets.example.test:443")
	require.ErrorIs(t, err, dialErr)
	require.Equal(t, "8.8.8.8:443", dialAddress)
}

func TestPublicAssetIPPolicyRejectsIANASpecialPurposeSpace(t *testing.T) {
	denied := []string{
		"0.0.0.1", "10.1.2.3", "100.64.0.1", "127.0.0.1", "169.254.1.1", "172.16.0.1",
		"192.0.0.9", "192.0.2.1", "192.31.196.1", "192.52.193.1", "192.88.99.1", "192.168.1.1",
		"192.175.48.1", "198.18.0.1", "198.51.100.1", "203.0.113.1", "224.0.0.1", "240.0.0.1",
		"255.255.255.255", "::", "::1", "::ffff:8.8.8.8", "::ffff:100.64.0.1", "64:ff9b::808:808",
		"64:ff9b:1::1", "100::1", "100:0:0:1::1", "2001::1", "2001:db8::1", "2002::1", "2620:4f:8000::1",
		"3fff::1", "5f00::1", "fc00::1", "fe80::1", "ff00::1",
	}
	for _, raw := range denied {
		t.Run(raw, func(t *testing.T) {
			require.False(t, isPublicAssetAddr(netip.MustParseAddr(raw)), raw)
		})
	}
	for _, raw := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		t.Run("public_"+raw, func(t *testing.T) {
			require.True(t, isPublicAssetAddr(netip.MustParseAddr(raw)), raw)
		})
	}
}

func TestAssetSourceLiteralAndEveryRedirectRejectSpecialPurposeAddresses(t *testing.T) {
	client := newPublicAssetHTTPClient(5 * time.Second)
	for _, raw := range []string{"100.64.0.1", "198.18.0.1", "192.0.2.1", "::ffff:8.8.8.8", "2001:db8::1"} {
		t.Run(raw, func(t *testing.T) {
			host := raw
			if strings.Contains(raw, ":") {
				host = "[" + raw + "]"
			}
			parsed, err := url.Parse("https://" + host + "/result.mp4")
			require.NoError(t, err)
			require.Error(t, validateAssetSourceURL(parsed), "literal must fail")

			request, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
			require.NoError(t, err)
			require.Error(t, client.CheckRedirect(request, []*http.Request{{URL: &url.URL{Scheme: "https", Host: "assets.example.test"}}}), "redirect hop must fail")
		})
	}
}

func TestHTTPAssetInspectorRejectsUnverifiableOrUnsafeSources(t *testing.T) {
	requestCount := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount++
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{"Content-Type": []string{"video/mp4"}},
			ContentLength: -1,
			Body:          io.NopCloser(strings.NewReader("")),
			Request:       request,
		}, nil
	})}
	inspector := newHTTPAssetInspector(client)

	_, err := inspector.Inspect(context.Background(), "http://127.0.0.1/private.mp4")
	require.Error(t, err)
	require.Zero(t, requestCount)

	_, err = inspector.Inspect(context.Background(), "https://assets.example.test/unknown.mp4")
	require.Error(t, err)
	require.Equal(t, 1, requestCount)
}
