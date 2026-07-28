package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *GatewayService) ForwardHCAtomSyncImage(ctx context.Context, c *gin.Context, account *Account, body []byte) (*ForwardResult, error) {
	started := time.Now()
	if s == nil || s.cfg == nil || !s.cfg.HCAtom.SyncImageEnabled {
		return nil, errors.New("HC-ATOM sync image dispatch is disabled")
	}
	var request struct {
		Model string `json:"model"`
		N     int    `json:"n"`
		Size  string `json:"size"`
	}
	if json.Unmarshal(body, &request) != nil {
		return nil, errors.New("HC-ATOM sync image request is invalid")
	}
	if _, ok := LookupHCAtomModel(HCAtomCapabilityImageSync, request.Model); !ok {
		return nil, errors.New("HC-ATOM sync image model is not enabled")
	}
	cipher, err := NewHCAtomCredentialCipher(s.cfg.BatchImage.HCAtomEncryptionKey)
	if err != nil {
		return nil, ErrHCAtomCredentialKeyUnavailable
	}
	key, err := ResolveHCAtomAPIKey(account, cipher)
	if err != nil {
		return nil, err
	}
	response, err := s.hcAtomFixedClient().Do(ctx, HCAtomCapabilityImageSync, request.Model, key, body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		status := response.StatusCode
		data, _ := readHCAtomResponse(response, key)
		if s.shouldFailoverUpstreamError(status) {
			return nil, &UpstreamFailoverError{StatusCode: status, ResponseBody: data}
		}
		return nil, errors.New("HC-ATOM sync image upstream rejected the request")
	}
	requestID := response.Header.Get("x-request-id")
	data, err := readHCAtomResponse(response, key)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || len(envelope.Data) == 0 {
		return nil, errors.New("HC-ATOM sync image response is invalid")
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	imageCount := len(envelope.Data)
	if request.N > 0 && imageCount > request.N {
		imageCount = request.N
	}
	size := strings.TrimSpace(request.Size)
	spec, _ := LookupHCAtomModel(HCAtomCapabilityImageSync, request.Model)
	return &ForwardResult{RequestID: requestID, Model: request.Model, UpstreamModel: spec.UpstreamModel, Duration: time.Since(started), ImageCount: imageCount, ImageSize: size, ImageOutputSize: size}, nil
}
