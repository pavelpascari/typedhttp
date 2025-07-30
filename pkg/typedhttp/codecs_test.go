package typedhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestCodecRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=0,max=120"`
}

type TestQueryRequest struct {
	Query  string `query:"q" validate:"required"`
	Limit  int    `query:"limit" default:"10" validate:"min=1,max=100"`
	Offset int    `query:"offset" default:"0" validate:"min=0"`
}

func TestJSONDecoder_Success(t *testing.T) {
	decoder := typedhttp.NewJSONDecoder[TestCodecRequest](validator.New())

	requestBody := TestCodecRequest{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	jsonData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	result, err := decoder.Decode(req)

	require.NoError(t, err)
	assert.Equal(t, requestBody.Name, result.Name)
	assert.Equal(t, requestBody.Email, result.Email)
	assert.Equal(t, requestBody.Age, result.Age)
}

func TestJSONDecoder_ValidationError(t *testing.T) {
	decoder := typedhttp.NewJSONDecoder[TestCodecRequest](validator.New())

	requestBody := TestCodecRequest{
		Name:  "",              // Invalid: required field is empty
		Email: "invalid-email", // Invalid: not a valid email
		Age:   -5,              // Invalid: below minimum
	}

	jsonData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	_, err = decoder.Decode(req)

	require.Error(t, err)

	var valErr *typedhttp.ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.Equal(t, "Validation failed", valErr.Message)
	assert.NotEmpty(t, valErr.Fields)
}

func TestJSONDecoder_InvalidJSON(t *testing.T) {
	decoder := typedhttp.NewJSONDecoder[TestCodecRequest](validator.New())

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	_, err := decoder.Decode(req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestJSONDecoder_ContentTypes(t *testing.T) {
	decoder := typedhttp.NewJSONDecoder[TestCodecRequest](nil)

	contentTypes := decoder.ContentTypes()

	assert.Equal(t, []string{"application/json"}, contentTypes)
}

func TestQueryDecoder_Success(t *testing.T) {
	decoder := typedhttp.NewQueryDecoder[TestQueryRequest](validator.New())

	req := httptest.NewRequest(http.MethodGet, "/test?q=search&limit=20&offset=5", http.NoBody)

	result, err := decoder.Decode(req)

	require.NoError(t, err)
	assert.Equal(t, "search", result.Query)
	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 5, result.Offset)
}

func TestQueryDecoder_DefaultValues(t *testing.T) {
	decoder := typedhttp.NewQueryDecoder[TestQueryRequest](validator.New())

	req := httptest.NewRequest(http.MethodGet, "/test?q=search", http.NoBody)

	result, err := decoder.Decode(req)

	require.NoError(t, err)
	assert.Equal(t, "search", result.Query)
	assert.Equal(t, 10, result.Limit) // Default value
	assert.Equal(t, 0, result.Offset) // Default value
}

func TestQueryDecoder_ValidationError(t *testing.T) {
	decoder := typedhttp.NewQueryDecoder[TestQueryRequest](validator.New())

	req := httptest.NewRequest(http.MethodGet, "/test?limit=200", http.NoBody) // Missing required 'q', limit too high

	_, err := decoder.Decode(req)

	require.Error(t, err)

	var valErr *typedhttp.ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.Equal(t, "Validation failed", valErr.Message)
}

func TestQueryDecoder_ContentTypes(t *testing.T) {
	decoder := typedhttp.NewQueryDecoder[TestQueryRequest](nil)

	contentTypes := decoder.ContentTypes()

	assert.Equal(t, []string{"application/x-www-form-urlencoded"}, contentTypes)
}

func TestJSONEncoder_Success(t *testing.T) {
	encoder := typedhttp.NewJSONEncoder[TestCodecRequest]()

	response := TestCodecRequest{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		Age:   25,
	}

	w := httptest.NewRecorder()

	err := encoder.Encode(w, response, http.StatusOK)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result TestCodecRequest
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, response, result)
}

func TestJSONEncoder_ContentType(t *testing.T) {
	encoder := typedhttp.NewJSONEncoder[TestCodecRequest]()

	contentType := encoder.ContentType()

	assert.Equal(t, "application/json", contentType)
}

func TestEnvelopeEncoder_Success(t *testing.T) {
	jsonEncoder := typedhttp.NewJSONEncoder[typedhttp.EnvelopeResponse[TestCodecRequest]]()
	envelopeEncoder := typedhttp.NewEnvelopeEncoder(jsonEncoder)

	response := TestCodecRequest{
		Name:  "Bob Smith",
		Email: "bob@example.com",
		Age:   35,
	}

	w := httptest.NewRecorder()
	w.Header().Set("X-Request-ID", "test-request-id")

	err := envelopeEncoder.Encode(w, response, http.StatusCreated)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var envelope typedhttp.EnvelopeResponse[TestCodecRequest]
	err = json.Unmarshal(w.Body.Bytes(), &envelope)
	require.NoError(t, err)

	assert.Equal(t, response, envelope.Data)
	assert.Equal(t, "test-request-id", envelope.RequestID)
	assert.Empty(t, envelope.Error)
}

func TestEnvelopeEncoder_ContentType(t *testing.T) {
	jsonEncoder := typedhttp.NewJSONEncoder[typedhttp.EnvelopeResponse[string]]()
	envelopeEncoder := typedhttp.NewEnvelopeEncoder(jsonEncoder)

	contentType := envelopeEncoder.ContentType()

	assert.Equal(t, "application/json", contentType)
}

// Test setFieldValue function indirectly through QueryDecoder.
func TestQueryDecoder_FieldTypes(t *testing.T) {
	type AllTypesRequest struct {
		StringField string  `query:"str"`
		IntField    int     `query:"int"`
		FloatField  float64 `query:"float"`
		BoolField   bool    `query:"bool"`
		UintField   uint    `query:"uint"`
	}

	decoder := typedhttp.NewQueryDecoder[AllTypesRequest](nil)

	// Build query string
	values := url.Values{}
	values.Set("str", "test")
	values.Set("int", "42")
	values.Set("float", "3.14")
	values.Set("bool", "true")
	values.Set("uint", "100")

	req := httptest.NewRequest(http.MethodGet, "/test?"+values.Encode(), http.NoBody)

	result, err := decoder.Decode(req)

	require.NoError(t, err)
	assert.Equal(t, "test", result.StringField)
	assert.Equal(t, 42, result.IntField)
	assert.Equal(t, 3.14, result.FloatField)
	assert.Equal(t, true, result.BoolField)
	assert.Equal(t, uint(100), result.UintField)
}
