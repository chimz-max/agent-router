// Copyright Envoy AI Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package translator

import (
	stdjson "encoding/json" //nolint:depguard // Sparse decoding must skip vectors without Sonic's per-element parser tables.
	"fmt"
	"io"
	"path"
	"strconv"

	"github.com/tidwall/sjson"

	"github.com/envoyproxy/ai-gateway/internal/apischema/openai"
	"github.com/envoyproxy/ai-gateway/internal/internalapi"
	"github.com/envoyproxy/ai-gateway/internal/json"
	"github.com/envoyproxy/ai-gateway/internal/metrics"
	"github.com/envoyproxy/ai-gateway/internal/tracing/tracingapi"
)

// NewEmbeddingOpenAIToOpenAITranslator implements [Factory] for OpenAI to OpenAI translation for embeddings.
func NewEmbeddingOpenAIToOpenAITranslator(apiVersion string, modelNameOverride internalapi.ModelNameOverride) OpenAIEmbeddingTranslator {
	return &openAIToOpenAITranslatorV1Embedding{modelNameOverride: modelNameOverride, path: path.Join("/", apiVersion, "embeddings")}
}

// openAIToOpenAITranslatorV1Embedding is a passthrough translator for OpenAI Embeddings API.
// May apply model overrides but otherwise preserves the OpenAI format:
// https://platform.openai.com/docs/api-reference/embeddings/create
type openAIToOpenAITranslatorV1Embedding struct {
	modelNameOverride internalapi.ModelNameOverride
	// The path of the embeddings endpoint to be used for the request. It is prefixed with the OpenAI path prefix.
	path string
}

// RequestBody implements [OpenAIEmbeddingTranslator.RequestBody].
func (o *openAIToOpenAITranslatorV1Embedding) RequestBody(original []byte, _ *openai.EmbeddingRequest, onRetry bool) (
	newHeaders []internalapi.Header, newBody []byte, err error,
) {
	if o.modelNameOverride != "" {
		// If modelName is set we override the model to be used for the request.
		newBody, err = sjson.SetBytesOptions(original, "model", o.modelNameOverride, sjsonOptions)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to set model name: %w", err)
		}
	}

	// Always set the path header to the embeddings endpoint so that the request is routed correctly.
	if onRetry && len(newBody) == 0 {
		newBody = original
	}
	newHeaders = []internalapi.Header{{pathHeaderName, o.path}}

	if len(newBody) > 0 {
		newHeaders = append(newHeaders, internalapi.Header{contentLengthHeaderName, strconv.Itoa(len(newBody))})
	}
	return
}

// ResponseHeaders implements [OpenAIEmbeddingTranslator.ResponseHeaders].
func (o *openAIToOpenAITranslatorV1Embedding) ResponseHeaders(map[string]string) (newHeaders []internalapi.Header, err error) {
	return nil, nil
}

type embeddingResponseMetadata struct {
	Model internalapi.ResponseModel `json:"model"`
	Usage openai.EmbeddingUsage     `json:"usage"`
}

// ignoredJSONValue lets the standard decoder validate and skip an unknown
// field without allocating its contents or materializing embedding vectors.
type ignoredJSONValue struct{}

func (*ignoredJSONValue) UnmarshalJSON([]byte) error { return nil }

func decodeEmbeddingResponseMetadata(body io.Reader) (metadata embeddingResponseMetadata, err error) {
	decoder := stdjson.NewDecoder(body)
	start, err := decoder.Token()
	if err != nil || start == nil {
		return metadata, err
	}
	if start != stdjson.Delim('{') {
		return metadata, fmt.Errorf("expected an embeddings response object")
	}
	for decoder.More() {
		var key stdjson.Token
		key, err = decoder.Token()
		if err != nil {
			return metadata, err
		}
		// Match the case-sensitive upstream contract; decoding into a struct
		// would also accept MODEL, USAGE and other case-insensitive aliases.
		switch key {
		case "model":
			err = decoder.Decode(&metadata.Model)
		case "usage":
			err = decodeEmbeddingUsage(decoder, &metadata.Usage)
		default:
			err = decoder.Decode(&ignoredJSONValue{})
		}
		if err != nil {
			return metadata, err
		}
	}
	_, err = decoder.Token()
	return metadata, err
}

func decodeEmbeddingUsage(decoder *stdjson.Decoder, usage *openai.EmbeddingUsage) error {
	start, err := decoder.Token()
	if err != nil || start == nil {
		return err
	}
	if start != stdjson.Delim('{') {
		return fmt.Errorf("expected an embeddings usage object")
	}
	for decoder.More() {
		var key stdjson.Token
		key, err = decoder.Token()
		if err != nil {
			return err
		}
		switch key {
		case "prompt_tokens":
			err = decoder.Decode(&usage.PromptTokens)
		case "total_tokens":
			err = decoder.Decode(&usage.TotalTokens)
		default:
			err = decoder.Decode(&ignoredJSONValue{})
		}
		if err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

// ResponseBody implements [OpenAIEmbeddingTranslator.ResponseBody].
// OpenAI embeddings support model virtualization through automatic routing and resolution,
// so we return the actual model from the response body which may differ from the requested model
// (e.g., request "text-embedding-3-small" → response with specific version).
func (o *openAIToOpenAITranslatorV1Embedding) ResponseBody(_ map[string]string, body io.Reader, _ bool, span tracingapi.EmbeddingsSpan) (
	newHeaders []internalapi.Header, newBody []byte, tokenUsage metrics.TokenUsage, responseModel internalapi.ResponseModel, err error,
) {
	var resp openai.EmbeddingResponse
	if span != nil {
		// Recording spans may capture vectors, so preserve the complete tracing response.
		if err := json.NewDecoder(body).Decode(&resp); err != nil {
			return nil, nil, tokenUsage, "", fmt.Errorf("failed to unmarshal body: %w", err)
		}
		span.RecordResponse(&resp)
	} else {
		// Passthrough only needs model and usage. The standard decoder validates
		// JSON while skipping data; Sonic builds a large parser table even for
		// omitted fields, then EmbeddingUnion decodes the vector a second time.
		metadata, err := decodeEmbeddingResponseMetadata(body)
		if err != nil {
			return nil, nil, tokenUsage, "", fmt.Errorf("failed to unmarshal body: %w", err)
		}
		resp.Model, resp.Usage = metadata.Model, metadata.Usage
	}

	// Embeddings don't return output tokens; populate input and total when provided.
	tokenUsage.SetInputTokens(uint32(resp.Usage.PromptTokens)) //nolint:gosec
	tokenUsage.SetTotalTokens(uint32(resp.Usage.TotalTokens))  //nolint:gosec
	responseModel = resp.Model
	return
}

// ResponseError implements [Translator.ResponseError]
func (o *openAIToOpenAITranslatorV1Embedding) ResponseError(respHeaders map[string]string, body io.Reader) ([]internalapi.Header, []byte, error) {
	return convertErrorOpenAIToOpenAIError(respHeaders, body)
}
