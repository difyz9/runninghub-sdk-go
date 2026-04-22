package runninghub

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

func normalizeOpenAPIV2Path(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", errors.New("empty path")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if strings.HasPrefix(p, "/openapi/") {
		return p, nil
	}
	return "/openapi/v2" + p, nil
}

// RunStandardModel submits a task to any "标准模型API" endpoint.
//
// Pass either:
// - a full path like "/openapi/v2/vidu/image-to-video-q3-pro-fast", or
// - a relative v2 path like "vidu/image-to-video-q3-pro-fast".
//
// The response is a task object; use QueryTaskV2 to poll for results.
func (c *Client) RunStandardModel(ctx context.Context, path string, req any) (*QueryV2Response, error) {
	p, err := normalizeOpenAPIV2Path(path)
	if err != nil {
		return nil, err
	}

	var out QueryV2Response
	if err := c.doJSON(ctx, http.MethodPost, p, nil, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type PricePreviewResponse struct {
	EstimatedPrice          float64 `json:"estimatedPrice"`
	Currency                string  `json:"currency"`
	FreeLimit               bool    `json:"freeLimit"`
	IsFreeThisCall          bool    `json:"isFreeThisCall"`
	PriceText               string  `json:"priceText"`
	PriceTextEn             string  `json:"priceTextEn"`
	RemainingFreeLimitCount string  `json:"remainingFreeLimitCount"`
	FreeLimitCount          string  `json:"freeLimitCount"`
}

// PricePreview calls /openapi/v2/price-preview/** for a given model API path.
//
// The request body MUST match the corresponding model API body.
func (c *Client) PricePreview(ctx context.Context, modelPath string, req any) (*PricePreviewResponse, error) {
	p, err := normalizeOpenAPIV2Path(modelPath)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(p, "/openapi/v2/price-preview/") {
		// already a preview path
		var out PricePreviewResponse
		if err := c.doJSON(ctx, http.MethodPost, p, nil, nil, req, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}

	trimmed := strings.TrimPrefix(p, "/openapi/v2/")
	previewPath := "/openapi/v2/price-preview/" + trimmed

	var out PricePreviewResponse
	if err := c.doJSON(ctx, http.MethodPost, previewPath, nil, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
