package typedhttp

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormDecoderBasicTypes tests all basic type conversions for form data.
func TestFormDecoderBasicTypes(t *testing.T) {
	type FormRequest struct {
		StringField  string  `form:"str"`
		IntField     int     `form:"int"`
		Int8Field    int8    `form:"int8"`
		Int16Field   int16   `form:"int16"`
		Int32Field   int32   `form:"int32"`
		Int64Field   int64   `form:"int64"`
		UintField    uint    `form:"uint"`
		Uint8Field   uint8   `form:"uint8"`
		Uint16Field  uint16  `form:"uint16"`
		Uint32Field  uint32  `form:"uint32"`
		Uint64Field  uint64  `form:"uint64"`
		Float32Field float32 `form:"float32"`
		Float64Field float64 `form:"float64"`
		BoolField    bool    `form:"bool"`
	}

	decoder := NewFormDecoder[FormRequest](nil)

	// Create URL-encoded form data
	formData := url.Values{}
	formData.Set("str", "hello")
	formData.Set("int", "42")
	formData.Set("int8", "127")
	formData.Set("int16", "32767")
	formData.Set("int32", "2147483647")
	formData.Set("int64", "9223372036854775807")
	formData.Set("uint", "42")
	formData.Set("uint8", "255")
	formData.Set("uint16", "65535")
	formData.Set("uint32", "4294967295")
	formData.Set("uint64", "18446744073709551615")
	formData.Set("float32", "3.14")
	formData.Set("float64", "2.718281828")
	formData.Set("bool", "true")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "hello", result.StringField)
	assert.Equal(t, 42, result.IntField)
	assert.Equal(t, int8(127), result.Int8Field)
	assert.Equal(t, int16(32767), result.Int16Field)
	assert.Equal(t, int32(2147483647), result.Int32Field)
	assert.Equal(t, int64(9223372036854775807), result.Int64Field)
	assert.Equal(t, uint(42), result.UintField)
	assert.Equal(t, uint8(255), result.Uint8Field)
	assert.Equal(t, uint16(65535), result.Uint16Field)
	assert.Equal(t, uint32(4294967295), result.Uint32Field)
	assert.Equal(t, uint64(18446744073709551615), result.Uint64Field)
	assert.Equal(t, float32(3.14), result.Float32Field)
	assert.Equal(t, 2.718281828, result.Float64Field)
	assert.True(t, result.BoolField)
}

// TestFormDecoderMultipartForm tests multipart form data handling.
func TestFormDecoderMultipartForm(t *testing.T) {
	type FormRequest struct {
		Name        string                  `form:"name"`
		Email       string                  `form:"email"`
		Age         int                     `form:"age"`
		Avatar      *multipart.FileHeader   `form:"avatar"`
		Attachments []*multipart.FileHeader `form:"attachments"`
	}

	decoder := NewFormDecoder[FormRequest](nil)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add text fields
	_ = writer.WriteField("name", "John Doe")
	_ = writer.WriteField("email", "john@example.com")
	_ = writer.WriteField("age", "30")

	// Add single file
	avatarWriter, err := writer.CreateFormFile("avatar", "avatar.jpg")
	require.NoError(t, err)
	_, _ = avatarWriter.Write([]byte("fake avatar image data"))

	// Add multiple files
	attachment1Writer, err := writer.CreateFormFile("attachments", "doc1.pdf")
	require.NoError(t, err)
	_, _ = attachment1Writer.Write([]byte("fake pdf data 1"))

	attachment2Writer, err := writer.CreateFormFile("attachments", "doc2.pdf")
	require.NoError(t, err)
	_, _ = attachment2Writer.Write([]byte("fake pdf data 2"))

	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, "john@example.com", result.Email)
	assert.Equal(t, 30, result.Age)
	assert.NotNil(t, result.Avatar)
	assert.Equal(t, "avatar.jpg", result.Avatar.Filename)
	assert.Len(t, result.Attachments, 2)
	assert.Equal(t, "doc1.pdf", result.Attachments[0].Filename)
	assert.Equal(t, "doc2.pdf", result.Attachments[1].Filename)
}

// TestFormDecoderFileUploadsDisabled tests form decoder with file uploads disabled.
func TestFormDecoderFileUploadsDisabled(t *testing.T) {
	type FormRequest struct {
		Name string                `form:"name"`
		File *multipart.FileHeader `form:"file"`
	}

	// Create decoder with file uploads disabled
	decoder := NewFormDecoderWithOptions[FormRequest](nil, MaxFormMemory, false)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("name", "John")
	fileWriter, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, _ = fileWriter.Write([]byte("test content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	result, err := decoder.Decode(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file uploads not allowed")
	assert.Equal(t, "John", result.Name) // Non-file fields are processed before file fields fail
}

// TestFormDecoderTransformations tests all transformation functions for form data.
func TestFormDecoderTransformations(t *testing.T) {
	type FormRequest struct {
		FirstIP   string `form:"forwarded_ips" transform:"first_ip"`
		LowerCase string `form:"upper_text" transform:"to_lower"`
		UpperCase string `form:"lower_text" transform:"to_upper"`
		Trimmed   string `form:"spaces" transform:"trim_space"`
		AdminFlag string `form:"user_role" transform:"is_admin"`
	}

	decoder := NewFormDecoder[FormRequest](nil)

	formData := url.Values{}
	formData.Set("forwarded_ips", "192.168.1.1, 10.0.0.1, 172.16.0.1")
	formData.Set("upper_text", "HELLO WORLD")
	formData.Set("lower_text", "hello world")
	formData.Set("spaces", "  trimmed  ")
	formData.Set("user_role", "admin")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "192.168.1.1", result.FirstIP)
	assert.Equal(t, "hello world", result.LowerCase)
	assert.Equal(t, "HELLO WORLD", result.UpperCase)
	assert.Equal(t, "trimmed", result.Trimmed)
	assert.Equal(t, "true", result.AdminFlag)
}

// TestFormDecoderFormats tests custom format parsing for form data.
func TestFormDecoderFormats(t *testing.T) {
	type FormRequest struct {
		UnixTime     time.Time `form:"unix_time" format:"unix"`
		RFC3339Time  time.Time `form:"rfc3339_time" format:"rfc3339"`
		DateOnly     time.Time `form:"date" format:"2006-01-02"`
		CustomFormat time.Time `form:"custom" format:"02/Jan/2006:15:04:05 -0700"`
	}

	decoder := NewFormDecoder[FormRequest](nil)

	formData := url.Values{}
	formData.Set("unix_time", "1703518245")
	formData.Set("rfc3339_time", "2023-12-25T15:30:45Z")
	formData.Set("date", "2023-12-25")
	formData.Set("custom", "25/Dec/2023:15:30:45 +0000")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	// Verify unix timestamp
	expectedUnix := time.Unix(1703518245, 0).UTC()
	assert.True(t, expectedUnix.Equal(result.UnixTime.UTC()))

	// Verify RFC3339
	expectedRFC3339, _ := time.Parse(time.RFC3339, "2023-12-25T15:30:45Z")
	assert.True(t, expectedRFC3339.Equal(result.RFC3339Time))

	// Verify date only
	expectedDate, _ := time.Parse("2006-01-02", "2023-12-25")
	assert.True(t, expectedDate.Equal(result.DateOnly))

	// Verify custom format
	expectedCustom, _ := time.Parse("02/Jan/2006:15:04:05 -0700", "25/Dec/2023:15:30:45 +0000")
	assert.True(t, expectedCustom.Equal(result.CustomFormat))
}

// TestFormDecoderStringSlices tests comma-separated string slice handling.
func TestFormDecoderStringSlices(t *testing.T) {
	type FormRequest struct {
		Tags       []string `form:"tags"`
		Categories []string `form:"categories"`
		Empty      []string `form:"empty"`
	}

	decoder := NewFormDecoder[FormRequest](nil)

	formData := url.Values{}
	formData.Set("tags", "go, programming, web, api")
	formData.Set("categories", "tech,tutorial,  development ")
	formData.Set("empty", "")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, []string{"go", "programming", "web", "api"}, result.Tags)
	assert.Equal(t, []string{"tech", "tutorial", "development"}, result.Categories)
	assert.Nil(t, result.Empty) // Empty string field gets no value
}

// TestFormDecoderJSONFields tests JSON field parsing in forms.
func TestFormDecoderJSONFields(t *testing.T) {
	type UserPreferences struct {
		Theme       string   `json:"theme"`
		Language    string   `json:"language"`
		Timezone    string   `json:"timezone"`
		Permissions []string `json:"permissions"`
	}

	type FormRequest struct {
		Name        string                 `form:"name"`
		Preferences UserPreferences        `form:"preferences" json_field:"true"`
		Metadata    map[string]interface{} `form:"metadata"`
	}

	decoder := NewFormDecoder[FormRequest](nil)

	prefsJSON := `{"theme":"dark","language":"en","timezone":"UTC","permissions":["read","write"]}`
	metadataJSON := `{"version":"1.0","beta":true,"priority":5}`

	formData := url.Values{}
	formData.Set("name", "John Doe")
	formData.Set("preferences", prefsJSON)
	formData.Set("metadata", metadataJSON)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, "dark", result.Preferences.Theme)
	assert.Equal(t, "en", result.Preferences.Language)
	assert.Equal(t, "UTC", result.Preferences.Timezone)
	assert.Equal(t, []string{"read", "write"}, result.Preferences.Permissions)
	assert.Equal(t, "1.0", result.Metadata["version"])
	assert.Equal(t, true, result.Metadata["beta"])
	assert.Equal(t, float64(5), result.Metadata["priority"]) // JSON numbers become float64
}

// TestFormDecoderDefaultValues tests default value handling for form data.
func TestFormDecoderDefaultValues(t *testing.T) {
	type FormRequest struct {
		WithDefault    string    `form:"missing" default:"default-value"`
		WithNowDefault time.Time `form:"now_time" default:"now"`
		WithUUID       string    `form:"uuid_field" default:"generate_uuid"`
		NoDefault      string    `form:"none"`
	}

	decoder := NewFormDecoder[FormRequest](nil)

	// Request with no form data
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "default-value", result.WithDefault)
	assert.False(t, result.WithNowDefault.IsZero()) // Should be set to current time
	assert.Contains(t, result.WithUUID, "uuid-")    // Should contain uuid prefix
	assert.Equal(t, "", result.NoDefault)           // Should remain empty
}

// TestFormDecoderValidation tests validation integration for form data.
func TestFormDecoderValidation(t *testing.T) {
	type FormRequest struct {
		Name     string `form:"name" validate:"required,min=2"`
		Email    string `form:"email" validate:"required,email"`
		Age      int    `form:"age" validate:"min=18,max=100"`
		Password string `form:"password" validate:"required,min=8"`
	}

	validator := validator.New()
	decoder := NewFormDecoder[FormRequest](validator)

	tests := []struct {
		name        string
		formData    map[string]string
		shouldError bool
		errorField  string
	}{
		{
			name: "valid form data",
			formData: map[string]string{
				"name":     "John Doe",
				"email":    "john@example.com",
				"age":      "25",
				"password": "securepassword123",
			},
			shouldError: false,
		},
		{
			name: "missing required name",
			formData: map[string]string{
				"email":    "john@example.com",
				"age":      "25",
				"password": "securepassword123",
			},
			shouldError: true,
			errorField:  "required",
		},
		{
			name: "invalid email",
			formData: map[string]string{
				"name":     "John Doe",
				"email":    "invalid-email",
				"age":      "25",
				"password": "securepassword123",
			},
			shouldError: true,
			errorField:  "email",
		},
		{
			name: "age too young",
			formData: map[string]string{
				"name":     "John Doe",
				"email":    "john@example.com",
				"age":      "16",
				"password": "securepassword123",
			},
			shouldError: true,
			errorField:  "min",
		},
		{
			name: "password too short",
			formData: map[string]string{
				"name":     "John Doe",
				"email":    "john@example.com",
				"age":      "25",
				"password": "short",
			},
			shouldError: true,
			errorField:  "min",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{}
			for key, value := range tt.formData {
				formData.Set(key, value)
			}

			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			result, err := decoder.Decode(req)

			if tt.shouldError {
				require.Error(t, err)
				var validationErr *ValidationError
				assert.True(t, errors.As(err, &validationErr))
				assert.Contains(t, err.Error(), "Form validation failed")
				// Check that the error fields contain the expected validation tag
				found := false
				for _, field := range validationErr.Fields {
					if field == tt.errorField {
						found = true

						break
					}
				}
				assert.True(t, found, "Expected validation error for field: %s", tt.errorField)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.Name)
			}
		})
	}
}

// TestFormDecoderErrorHandling tests various error conditions for form data.
func TestFormDecoderErrorHandling(t *testing.T) {
	// Test invalid JSON in form field
	t.Run("invalid JSON in form field", func(t *testing.T) {
		type JSONRequest struct {
			Data map[string]interface{} `form:"json_field" json_field:"true"`
		}
		decoder := NewFormDecoder[JSONRequest](nil)

		formData := url.Values{}
		formData.Set("json_field", "{invalid json")
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		_, err := decoder.Decode(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})

	// Test invalid integer in form
	t.Run("invalid integer in form", func(t *testing.T) {
		type IntRequest struct {
			Number int `form:"int_field"`
		}
		decoder := NewFormDecoder[IntRequest](nil)

		formData := url.Values{}
		formData.Set("int_field", "not-a-number")
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		_, err := decoder.Decode(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integer value")
	})

	// Test unknown transformation
	t.Run("unknown transformation", func(t *testing.T) {
		type TransformRequest struct {
			Value string `form:"transform_field" transform:"unknown"`
		}
		decoder := NewFormDecoder[TransformRequest](nil)

		formData := url.Values{}
		formData.Set("transform_field", "value")
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		_, err := decoder.Decode(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown transformation")
	})
}

// TestFormDecoderHelperFunctions tests all form helper functions.
func TestFormDecoderHelperFunctions(t *testing.T) {
	// Setup form data
	formData := url.Values{}
	formData.Set("name", "John Doe")
	formData.Set("email", "john@example.com")
	formData.Add("tags", "go")
	formData.Add("tags", "programming")
	formData.Add("tags", "web")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	// Test GetFormValue
	name := GetFormValue(req, "name", "default")
	assert.Equal(t, "John Doe", name)

	missing := GetFormValue(req, "missing", "fallback")
	assert.Equal(t, "fallback", missing)

	// Test GetFormValues (for multiple values)
	tags := GetFormValues(req, "tags")
	assert.Equal(t, []string{"go", "programming", "web"}, tags)

	emptyTags := GetFormValues(req, "nonexistent")
	assert.Nil(t, emptyTags)

	// Test with no form parsed
	reqNoParse := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
	defaultValue := GetFormValue(reqNoParse, "name", "default")
	assert.Equal(t, "default", defaultValue)

	nilValues := GetFormValues(reqNoParse, "tags")
	assert.Nil(t, nilValues)
}

// TestFormDecoderFileHelperFunctions tests file-related helper functions.
func TestFormDecoderFileHelperFunctions(t *testing.T) {
	// Create multipart form with files
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add single file
	fileWriter, err := writer.CreateFormFile("single_file", "test.txt")
	require.NoError(t, err)
	_, _ = fileWriter.Write([]byte("test content"))

	// Add multiple files
	file1Writer, err := writer.CreateFormFile("multiple_files", "file1.txt")
	require.NoError(t, err)
	_, _ = file1Writer.Write([]byte("content 1"))

	file2Writer, err := writer.CreateFormFile("multiple_files", "file2.txt")
	require.NoError(t, err)
	_, _ = file2Writer.Write([]byte("content 2"))

	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(MaxFormMemory)

	// Test GetFileHeader
	fileHeader, err := GetFileHeader(req, "single_file")
	require.NoError(t, err)
	assert.Equal(t, "test.txt", fileHeader.Filename)
	assert.Greater(t, fileHeader.Size, int64(0))

	// Test GetFileHeader with missing file
	_, err = GetFileHeader(req, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no file found")

	// Test GetFileHeaders
	fileHeaders, err := GetFileHeaders(req, "multiple_files")
	require.NoError(t, err)
	assert.Len(t, fileHeaders, 2)
	assert.Equal(t, "file1.txt", fileHeaders[0].Filename)
	assert.Equal(t, "file2.txt", fileHeaders[1].Filename)

	// Test GetFileHeaders with missing files
	_, err = GetFileHeaders(req, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no files found")

	// Test with no multipart form
	reqNoMultipart := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	_, err = GetFileHeader(reqNoMultipart, "file")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no multipart form data")

	_, err = GetFileHeaders(reqNoMultipart, "files")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no multipart form data")
}

// TestFormDecoderFormInfo tests GetFormInfo function.
func TestFormDecoderFormInfo(t *testing.T) {
	// Create multipart form with both fields and files
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("name", "John Doe")
	_ = writer.WriteField("email", "john@example.com")

	fileWriter, err := writer.CreateFormFile("avatar", "avatar.jpg")
	require.NoError(t, err)
	_, _ = fileWriter.Write([]byte("fake image data"))

	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(MaxFormMemory)

	info := GetFormInfo(req)

	assert.Contains(t, info.ContentType, "multipart/form-data")
	assert.Equal(t, 2, info.FieldCount) // name and email
	assert.Equal(t, 1, info.FileCount)  // avatar
	assert.Greater(t, info.TotalSize, int64(0))
	assert.Equal(t, "John Doe", info.Fields["name"][0])
	assert.Equal(t, "john@example.com", info.Fields["email"][0])
	assert.Equal(t, "avatar.jpg", info.Files["avatar"][0].Filename)
}

// TestFormDecoderFileValidation tests ValidateFileUpload function.
func TestFormDecoderFileValidation(t *testing.T) {
	// Create a mock file header
	fileHeader := &multipart.FileHeader{
		Filename: "test.jpg",
		Size:     1024,
		Header:   make(map[string][]string),
	}
	fileHeader.Header.Set("Content-Type", "image/jpeg")

	tests := []struct {
		name        string
		options     FormOptions
		shouldError bool
		errorType   error
	}{
		{
			name: "valid file",
			options: FormOptions{
				AllowFiles:   true,
				MaxFileSize:  2048,
				AllowedTypes: []string{"image/jpeg", "image/png"},
			},
			shouldError: false,
		},
		{
			name: "files not allowed",
			options: FormOptions{
				AllowFiles: false,
			},
			shouldError: true,
			errorType:   ErrFileUploadsNotAllowed,
		},
		{
			name: "file too large",
			options: FormOptions{
				AllowFiles:  true,
				MaxFileSize: 512, // Smaller than file size (1024)
			},
			shouldError: true,
			errorType:   ErrFileTooLarge,
		},
		{
			name: "invalid file type",
			options: FormOptions{
				AllowFiles:   true,
				AllowedTypes: []string{"text/plain", "application/pdf"}, // Doesn't include image/jpeg
			},
			shouldError: true,
			errorType:   ErrInvalidFileType,
		},
		{
			name: "no file size limit",
			options: FormOptions{
				AllowFiles:  true,
				MaxFileSize: 0, // No limit
			},
			shouldError: false,
		},
		{
			name: "no allowed types restriction",
			options: FormOptions{
				AllowFiles:   true,
				AllowedTypes: []string{}, // No restriction
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileUpload(fileHeader, tt.options)

			if tt.shouldError {
				require.Error(t, err)
				if tt.errorType != nil {
					assert.True(t, errors.Is(err, tt.errorType))
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestFormDecoderContentTypes tests ContentTypes method.
func TestFormDecoderContentTypes(t *testing.T) {
	type FormRequest struct {
		Value string `form:"test"`
	}

	decoder := NewFormDecoder[FormRequest](nil)
	contentTypes := decoder.ContentTypes()

	assert.Len(t, contentTypes, 2)
	assert.Contains(t, contentTypes, "application/x-www-form-urlencoded")
	assert.Contains(t, contentTypes, "multipart/form-data")
}

// TestFormDecoderCombinedFeatures tests complex scenarios with multiple features.
func TestFormDecoderCombinedFeatures(t *testing.T) {
	type UserProfile struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	type ComplexFormRequest struct {
		// Basic fields
		Username string `form:"username"`
		Email    string `form:"email" validate:"required,email"`

		// With transformation
		NormalizedRole string `form:"role" transform:"to_lower"`

		// With format
		JoinDate time.Time `form:"join_date" format:"2006-01-02"`

		// JSON field
		Profile UserProfile `form:"profile" json_field:"true"`

		// String slice
		Skills []string `form:"skills"`

		// File upload
		Avatar *multipart.FileHeader `form:"avatar"`

		// With default
		Status string `form:"status" default:"active"`
	}

	val := validator.New()
	decoder := NewFormDecoder[ComplexFormRequest](val)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("username", "johndoe")
	_ = writer.WriteField("email", "john@example.com")
	_ = writer.WriteField("role", "ADMIN")
	_ = writer.WriteField("join_date", "2023-01-15")
	_ = writer.WriteField("profile", `{"name":"John Doe","age":30}`)
	_ = writer.WriteField("skills", "go, javascript, python")
	// status field intentionally omitted to test default

	avatarWriter, err := writer.CreateFormFile("avatar", "profile.jpg")
	require.NoError(t, err)
	_, _ = avatarWriter.Write([]byte("fake image data"))

	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "johndoe", result.Username)
	assert.Equal(t, "john@example.com", result.Email)
	assert.Equal(t, "admin", result.NormalizedRole) // Transformed to lowercase

	expectedDate, _ := time.Parse("2006-01-02", "2023-01-15")
	assert.True(t, expectedDate.Equal(result.JoinDate))

	assert.Equal(t, "John Doe", result.Profile.Name)
	assert.Equal(t, 30, result.Profile.Age)
	assert.Equal(t, []string{"go", "javascript", "python"}, result.Skills)
	assert.Equal(t, "active", result.Status) // Default value
	assert.NotNil(t, result.Avatar)
	assert.Equal(t, "profile.jpg", result.Avatar.Filename)
}
