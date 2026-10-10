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
func NewEmbeddingOpenAIToOpenAITranslator(apiVersion string, modelNameOverride internalapi.ModelNameOverride, validateEmbeddingVectors, traceEmbeddingVectors bool) OpenAIEmbeddingTranslator {
	return &openAIToOpenAITranslatorV1Embedding{modelNameOverride: modelNameOverride, validateEmbeddingVectors: validateEmbeddingVectors, traceEmbeddingVectors: traceEmbeddingVectors, path: path.Join("/", apiVersion, "embeddings")}
}

// openAIToOpenAITranslatorV1Embedding is a passthrough translator for OpenAI Embeddings API.
// May apply model overrides but otherwise preserves the OpenAI format:
// https://platform.openai.com/docs/api-reference/embeddings/create
type openAIToOpenAITranslatorV1Embedding struct {
	modelNameOverride        internalapi.ModelNameOverride
	validateEmbeddingVectors bool
	traceEmbeddingVectors    bool
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

// embeddingTraceVector treats decoding failures as missing trace output. Syntax
// is validated by the enclosing decoder; vector schema is not a passthrough rule.
type embeddingTraceVector openai.EmbeddingUnion

func (vector *embeddingTraceVector) UnmarshalJSON(body []byte) error {
	var decoded openai.EmbeddingUnion
	if err := decoded.UnmarshalJSON(body); err != nil {
		vector.Value = nil
		return nil
	}
	*vector = embeddingTraceVector(decoded)
	return nil
}

type embeddingTraceEntry struct {
	openai.Embedding
	Vector embeddingTraceVector `json:"embedding"`
}

type embeddingTraceData []openai.Embedding

func (data *embeddingTraceData) UnmarshalJSON(body []byte) error {
	var decoded []embeddingTraceEntry
	if err := json.Unmarshal(body, &decoded); err != nil {
		// Invalid trace-only fields (such as index) must not reject the response.
		*data = nil
		return nil
	}
	*data = make([]openai.Embedding, len(decoded))
	for i, entry := range decoded {
		(*data)[i] = entry.Embedding
		(*data)[i].Embedding = openai.EmbeddingUnion(entry.Vector)
	}
	return nil
}

// The object label is trace-only metadata, not a passthrough validation rule.
type embeddingTraceObject string

func (object *embeddingTraceObject) UnmarshalJSON(body []byte) error {
	var decoded string
	if err := json.Unmarshal(body, &decoded); err != nil {
		*object = ""
		return nil
	}
	*object = embeddingTraceObject(decoded)
	return nil
}

func decodeEmbeddingResponseMetadata(body io.Reader, traceResponse *openai.EmbeddingResponse) (metadata embeddingResponseMetadata, err error) {
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
		case "data":
			if traceResponse == nil {
				err = decoder.Decode(&ignoredJSONValue{})
			} else {
				err = decoder.Decode((*embeddingTraceData)(&traceResponse.Data))
			}
		case "object":
			if traceResponse == nil {
				err = decoder.Decode(&ignoredJSONValue{})
			} else {
				err = decoder.Decode((*embeddingTraceObject)(&traceResponse.Object))
			}
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
	if o.validateEmbeddingVectors {
		// Preserve upstream schema validation. Full decoding materializes vectors
		// and has substantially higher memory cost than metadata-only passthrough.
		if err := json.NewDecoder(body).Decode(&resp); err != nil {
			return nil, nil, tokenUsage, "", fmt.Errorf("failed to unmarshal body: %w", err)
		}
	} else {
		var traceResponse *openai.EmbeddingResponse
		if span != nil && o.traceEmbeddingVectors {
			traceResponse = &resp
		}
		// Sampling must not change validation. Trace vector capture is best-effort.
		metadata, err := decodeEmbeddingResponseMetadata(body, traceResponse)
		if err != nil {
			return nil, nil, tokenUsage, "", fmt.Errorf("failed to unmarshal body: %w", err)
		}
		resp.Model, resp.Usage = metadata.Model, metadata.Usage
	}
	if span != nil {
		span.RecordResponse(&resp)
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
