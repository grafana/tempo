// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_io "io"
	_nethttp "net/http"
	_neturl "net/url"
	"os"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// OrganizationsApi service type
type OrganizationsApi datadog.Service

// UploadIdPMetadataOptionalParameters holds optional parameters for UploadIdPMetadata.
type UploadIdPMetadataOptionalParameters struct {
	IdpFile **os.File
}

// NewUploadIdPMetadataOptionalParameters creates an empty struct for parameters.
func NewUploadIdPMetadataOptionalParameters() *UploadIdPMetadataOptionalParameters {
	this := UploadIdPMetadataOptionalParameters{}
	return &this
}

// WithIdpFile sets the corresponding parameter name and returns the struct.
func (r *UploadIdPMetadataOptionalParameters) WithIdpFile(idpFile *os.File) *UploadIdPMetadataOptionalParameters {
	r.IdpFile = &idpFile
	return r
}

// UploadIdPMetadata Upload IdP metadata.
// Endpoint for uploading IdP metadata for SAML setup.
//
// Use this endpoint to upload or replace IdP metadata for SAML login configuration.
func (a *OrganizationsApi) UploadIdPMetadata(ctx _context.Context, o ...UploadIdPMetadataOptionalParameters) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodPost
		localVarPostBody   interface{}
		optionalParams     UploadIdPMetadataOptionalParameters
	)

	if len(o) > 1 {
		return nil, datadog.ReportError("only one argument of type UploadIdPMetadataOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.OrganizationsApi.UploadIdPMetadata")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/saml_configurations/idp_metadata"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Content-Type"] = "multipart/form-data"
	localVarHeaderParams["Accept"] = "*/*"

	formFile := datadog.FormFile{}
	formFile.FormFileName = "idp_file"
	var localVarFile *os.File
	if optionalParams.IdpFile != nil {
		localVarFile = *optionalParams.IdpFile
	}
	if localVarFile != nil {
		fbs, _ := _io.ReadAll(localVarFile)
		formFile.FileBytes = fbs
		formFile.FileName = localVarFile.Name()
		localVarFile.Close()
	}
	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, &formFile)
	if err != nil {
		return nil, err
	}

	localVarHTTPResponse, err := a.Client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarHTTPResponse, err
	}

	localVarBody, err := datadog.ReadBody(localVarHTTPResponse)
	if err != nil {
		return localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := datadog.GenericOpenAPIError{
			ErrorBody:    localVarBody,
			ErrorMessage: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.Client.Decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarHTTPResponse, newErr
			}
			newErr.ErrorModel = v
		}
		return localVarHTTPResponse, newErr
	}

	return localVarHTTPResponse, nil
}

// NewOrganizationsApi Returns NewOrganizationsApi.
func NewOrganizationsApi(client *datadog.APIClient) *OrganizationsApi {
	return &OrganizationsApi{
		Client: client,
	}
}
