package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

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
