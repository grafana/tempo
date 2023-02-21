// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

var (
	jsonCheck = regexp.MustCompile(`(?i:(?:application|text)/(?:vnd\.[^;]+\+)?json)`)
	xmlCheck  = regexp.MustCompile(`(?i:(?:application|text)/xml)`)
)

// APIClient manages communication with the Datadog API V2 Collection API v1.0.
// In most cases there should be only one, shared, APIClient.
type APIClient struct {
	Cfg *Configuration
}

// FormFile holds parameters for a file in multipart/form-data request.
type FormFile struct {
	FormFileName string
	FileName     string
	FileBytes    []byte
}

// Service holds APIClient
type Service struct {
	Client *APIClient
}

// SetAuthKeys sets the appropriate values in the headers parameter.
func SetAuthKeys(ctx context.Context, headerParams *map[string]string, keys ...[2]string) {
	if ctx != nil {
		for _, key := range keys {
			if auth, ok := ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
				if apiKey, ok := auth[key[0]]; ok {
					(*headerParams)[key[1]] = apiKey.Key
				}
			}
		}
	}
}

// ReadBody returns the byte content of the response and make it available again on the response object.
func ReadBody(response *http.Response) ([]byte, error) {
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	response.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, err
}

// NewAPIClient creates a new API client. Requires a userAgent string describing your application.
// optionally a custom http.Client to allow for advanced features such as caching.
func NewAPIClient(cfg *Configuration) *APIClient {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	c := &APIClient{}
	c.Cfg = cfg

	return c
}

// ParameterToString convert interface{} parameters to string, using a delimiter if format is provided.
func ParameterToString(obj interface{}, collectionFormat string) string {
	var delimiter string

	switch collectionFormat {
	case "pipes":
		delimiter = "|"
	case "ssv":
		delimiter = " "
	case "tsv":
		delimiter = "\t"
	case "csv":
		delimiter = ","
	}

	if reflect.TypeOf(obj).Kind() == reflect.Slice {
		return strings.Trim(strings.Replace(fmt.Sprint(obj), " ", delimiter, -1), "[]")
	} else if t, ok := obj.(time.Time); ok {
		if t.Nanosecond() == 0 {
			return t.Format("2006-01-02T15:04:05Z07:00")
		}
		return t.Format("2006-01-02T15:04:05.000Z07:00")
	}

	return fmt.Sprintf("%v", obj)
}

// CallAPI do the request.
func (c *APIClient) CallAPI(request *http.Request) (*http.Response, error) {
	if c.Cfg.Debug {
		dump, err := httputil.DumpRequestOut(request, true)
		if err != nil {
			return nil, err
		}
		// Strip any api keys from the response being logged
		keys, ok := request.Context().Value(ContextAPIKeys).(map[string]APIKey)
		if keys != nil && ok {
			for _, apiKey := range keys {
				valueRegex := regexp.MustCompile(fmt.Sprintf("(?m)%s", apiKey.Key))
				dump = valueRegex.ReplaceAll(dump, []byte("REDACTED"))
			}
		}
		log.Printf("\n%s\n", string(dump))
	}

	resp, err := c.Cfg.HTTPClient.Do(request)
	if err != nil {
		return resp, err
	}

	if c.Cfg.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return resp, err
		}
		log.Printf("\n%s\n", string(dump))
	}
	return resp, err
}

// GetConfig allows modification of underlying config for alternate implementations and testing.
// Caution: modifying the configuration while live can cause data races and potentially unwanted behavior.
func (c *APIClient) GetConfig() *Configuration {
	return c.Cfg
}

// PrepareRequest build the request.
func (c *APIClient) PrepareRequest(
	ctx context.Context,
	path string, method string,
	postBody interface{},
	headerParams map[string]string,
	queryParams url.Values,
	formParams url.Values,
	formFile *FormFile) (localVarRequest *http.Request, err error) {

	var body *bytes.Buffer

	// Detect postBody type and post.
	if postBody != nil {
		contentType := headerParams["Content-Type"]
		if contentType == "" {
			contentType = detectContentType(postBody)
			headerParams["Content-Type"] = contentType
		}

		body, err = setBody(postBody, contentType)
		if err != nil {
			return nil, err
		}
	}

	// add form parameters and file if available.
	if strings.HasPrefix(headerParams["Content-Type"], "multipart/form-data") && len(formParams) > 0 || formFile != nil {
		if body != nil {
			return nil, errors.New("cannot specify postBody and multipart form at the same time")
		}
		body = &bytes.Buffer{}
		w := multipart.NewWriter(body)

		for k, v := range formParams {
			for _, iv := range v {
				if strings.HasPrefix(k, "@") { // file
					err = addFile(w, k[1:], iv)
					if err != nil {
						return nil, err
					}
				} else { // form value
					w.WriteField(k, iv)
				}
			}
		}
		if formFile != nil {
			w.Boundary()
			part, err := w.CreateFormFile(formFile.FormFileName, filepath.Base(formFile.FileName))
			if err != nil {
				return nil, err
			}
			_, err = part.Write(formFile.FileBytes)
			if err != nil {
				return nil, err
			}
		}

		// Set the Boundary in the Content-Type
		headerParams["Content-Type"] = w.FormDataContentType()

		// Set Content-Length
		headerParams["Content-Length"] = fmt.Sprintf("%d", body.Len())
		w.Close()
	}

	if strings.HasPrefix(headerParams["Content-Type"], "application/x-www-form-urlencoded") && len(formParams) > 0 {
		if body != nil {
			return nil, errors.New("cannot specify postBody and x-www-form-urlencoded form at the same time")
		}
		body = &bytes.Buffer{}
		body.WriteString(formParams.Encode())
		// Set Content-Length
		headerParams["Content-Length"] = fmt.Sprintf("%d", body.Len())
	}

	// Setup path and query parameters
	url, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	// Override request host, if applicable
	if c.Cfg.Host != "" {
		url.Host = c.Cfg.Host
	}

	// Override request scheme, if applicable
	if c.Cfg.Scheme != "" {
		url.Scheme = c.Cfg.Scheme
	}

	// Adding Query Param
	query := url.Query()
	for k, v := range queryParams {
		for _, iv := range v {
			query.Add(k, iv)
		}
	}

	// Encode the parameters.
	url.RawQuery = query.Encode()

	// Generate a new request
	if body != nil {
		if headerParams["Content-Encoding"] == "gzip" {
			var buf bytes.Buffer
			compressor := gzip.NewWriter(&buf)
			if _, err = compressor.Write(body.Bytes()); err != nil {
				return nil, err
			}
			if err = compressor.Close(); err != nil {
				return nil, err
			}
			body = &buf

		} else if headerParams["Content-Encoding"] == "deflate" {
			var buf bytes.Buffer
			compressor := zlib.NewWriter(&buf)
			if _, err = compressor.Write(body.Bytes()); err != nil {
				return nil, err
			}
			if err = compressor.Close(); err != nil {
				return nil, err
			}
			body = &buf
		} else if headerParams["Content-Encoding"] == "zstd1" {
			body, err = compressZstd(body.Bytes())
			if err != nil {
				return nil, err
			}
		}
		headerParams["Content-Length"] = fmt.Sprintf("%d", body.Len())
		localVarRequest, err = http.NewRequest(method, url.String(), body)
	} else {
		localVarRequest, err = http.NewRequest(method, url.String(), nil)
	}
	if err != nil {
		return nil, err
	}

	// add header parameters, if any
	if len(headerParams) > 0 {
		headers := http.Header{}
		for h, v := range headerParams {
			headers.Set(h, v)
		}
		localVarRequest.Header = headers
	}

	// Add the user agent to the request.
	localVarRequest.Header.Add("User-Agent", c.Cfg.UserAgent)

	if ctx != nil {
		// add context to the request
		localVarRequest = localVarRequest.WithContext(ctx)

		// Walk through any authentication.

		// OAuth2 authentication
		if tok, ok := ctx.Value(ContextOAuth2).(oauth2.TokenSource); ok {
			// We were able to grab an oauth2 token from the context
			var latestToken *oauth2.Token
			if latestToken, err = tok.Token(); err != nil {
				return nil, err
			}

			latestToken.SetAuthHeader(localVarRequest)
		}

		// Basic HTTP Authentication
		if auth, ok := ctx.Value(ContextBasicAuth).(BasicAuth); ok {
			localVarRequest.SetBasicAuth(auth.UserName, auth.Password)
		}

		// AccessToken Authentication
		if auth, ok := ctx.Value(ContextAccessToken).(string); ok {
			localVarRequest.Header.Add("Authorization", "Bearer "+auth)
		}
	}

	for header, value := range c.Cfg.DefaultHeader {
		localVarRequest.Header.Add(header, value)
	}

	if !c.Cfg.Compress {
		// gzip is on by default, so disable it by setting encoding to identity
		localVarRequest.Header.Add("Accept-Encoding", "identity")
	}
	return localVarRequest, nil
}

// Decode unmarshal bytes into an interface
func (c *APIClient) Decode(v interface{}, b []byte, contentType string) (err error) {
	if len(b) == 0 {
		return nil
	}
	if s, ok := v.(*string); ok {
		*s = string(b)
		return nil
	}
	if xmlCheck.MatchString(contentType) {
		if err = xml.Unmarshal(b, v); err != nil {
			return err
		}
		return nil
	}
	if actualObj, ok := v.(interface{ GetActualInstance() interface{} }); ok { // oneOf, anyOf schemas
		if unmarshalObj, ok := actualObj.(interface{ UnmarshalJSON([]byte) error }); ok { // make sure it has UnmarshalJSON defined
			if err = unmarshalObj.UnmarshalJSON(b); err != nil {
				return err
			}
		} else {
			return errors.New("unknown type with GetActualInstance but no unmarshalObj.UnmarshalJSON defined")
		}
	} else if err = json.Unmarshal(b, v); err != nil { // simple model
		return err
	}
	return nil
}

// Add a file to the multipart request.
func addFile(w *multipart.Writer, fieldName, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := w.CreateFormFile(fieldName, filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)

	return err
}

// ReportError Prevent trying to import "fmt".
func ReportError(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}

// Set request body from an interface{}.
func setBody(body interface{}, contentType string) (bodyBuf *bytes.Buffer, err error) {
	if reflect.ValueOf(body).IsNil() {
		return nil, nil
	}

	if bodyBuf == nil {
		bodyBuf = &bytes.Buffer{}
	}

	if reader, ok := body.(io.Reader); ok {
		_, err = bodyBuf.ReadFrom(reader)
	} else if fp, ok := body.(**os.File); ok {
		_, err = bodyBuf.ReadFrom(*fp)
	} else if b, ok := body.([]byte); ok {
		_, err = bodyBuf.Write(b)
	} else if s, ok := body.(string); ok {
		_, err = bodyBuf.WriteString(s)
	} else if s, ok := body.(*string); ok {
		_, err = bodyBuf.WriteString(*s)
	} else if jsonCheck.MatchString(contentType) {
		err = json.NewEncoder(bodyBuf).Encode(body)
	} else if xmlCheck.MatchString(contentType) {
		err = xml.NewEncoder(bodyBuf).Encode(body)
	}

	if err != nil {
		return nil, err
	}

	if bodyBuf.Len() == 0 {
		return nil, fmt.Errorf("invalid body type %s", contentType)
	}
	return bodyBuf, nil
}

// detectContentType method is used to figure out `Request.Body` content type for request header.
func detectContentType(body interface{}) string {
	contentType := "text/plain; charset=utf-8"
	kind := reflect.TypeOf(body).Kind()

	switch kind {
	case reflect.Struct, reflect.Map, reflect.Ptr:
		contentType = "application/json; charset=utf-8"
	case reflect.String:
		contentType = "text/plain; charset=utf-8"
	default:
		if b, ok := body.([]byte); ok {
			contentType = http.DetectContentType(b)
		} else if kind == reflect.Slice {
			contentType = "application/json; charset=utf-8"
		}
	}

	return contentType
}

// GenericOpenAPIError Provides access to the body, error and model on returned errors.
type GenericOpenAPIError struct {
	ErrorBody    []byte
	ErrorMessage string
	ErrorModel   interface{}
}

// Error returns non-empty string if there was an error.
func (e GenericOpenAPIError) Error() string {
	return e.ErrorMessage
}

// Body returns the raw bytes of the response.
func (e GenericOpenAPIError) Body() []byte {
	return e.ErrorBody
}

// Model returns the unpacked model of the error.
func (e GenericOpenAPIError) Model() interface{} {
	return e.ErrorModel
}
