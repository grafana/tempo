// Package datadog-api-client-go.
//
// This repository contains a Go API client for the Datadog API (https://docs.datadoghq.com/api/).
//
// Requirements
//
// • Go 1.17+
//
// Layout
//
// This repository contains per-major-version API client packages. Right
// now, Datadog has two API versions,
// v1, v2 and the common package.
//
// The API v1 Client
//
// The client library for Datadog API v1 is located in the api/datadogV1 directory. Import it with
//
//   import "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
//
// The API v2 Client
//
// The client library for Datadog API v2 is located in the api/datadogV2 directory. Import it with
//
//   import "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
//
// The Datadog Package
//
// The datadog package for Datadog API is located in the api/datadog directory. Import it with
//
//   import "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
//
// Getting Started
//
// Here's an example creating a user:
//
//   package main
//
//   import (
//       "context"
//       "fmt"
//       "os"
//
//       "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
//       "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
//   )
//
//   func main() {
//       ctx := context.WithValue(
//           context.Background(),
//           datadog.ContextAPIKeys,
//           map[string]datadog.APIKey{
//               "apiKeyAuth": {
//                   Key: os.Getenv("DD_CLIENT_API_KEY"),
//               },
//               "appKeyAuth": {
//                   Key: os.Getenv("DD_CLIENT_APP_KEY"),
//               },
//           },
//       )
//
//       body := *datadogV2.NewUserCreateRequest(*datadogV2.NewUserCreateData(*datadogV2.NewUserCreateAttributes("jane.doe@example.com"), datadogV2.UsersType("users")))
//
//       configuration := datadog.NewConfiguration()
//       apiClient := datadog.NewAPIClient(configuration)
//       usersApi := datadogV2.NewUsersApi(apiClient)
//
//       resp, r, err := usersApi.CreateUser(ctx, body)
//       if err != nil {
//           fmt.Fprintf(os.Stderr, "Error creating user: %v\n", err)
//           fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
//       }
//       responseData := resp.GetData()
//       fmt.Fprintf(os.Stdout, "User ID: %s", responseData.GetId())
//   }
//
// Save it to example.go, then run go get github.com/DataDog/datadog-api-client-go/v2.
// Set the
// DD_CLIENT_API_KEY and DD_CLIENT_APP_KEY to your Datadog
// credentials, and then run
// go run example.go.
//
// Unstable Endpoints
//
// This client includes access to Datadog API endpoints while they are in an unstable state and may undergo breaking changes. An extra configuration step is required to enable these endpoints:
//
//       configuration.SetUnstableOperationEnabled("<APIVersion>.<OperationName>", true)
//
// where <OperationName> is the name of the method used to interact with that endpoint. For example: GetLogsIndex, or UpdateLogsIndex
//
// Changing Server
//
// When talking to a different server, like the eu instance, change the ContextServerVariables:
//
//       ctx = context.WithValue(ctx,
//           datadog.ContextServerVariables,
//           map[string]string{
//               "site": "datadoghq.eu",
//       })
//
// Disable compressed payloads
//
// If you want to disable GZIP compressed responses, set the compress flag
// on your configuration object:
//
//
//       configuration.Compress = false
//
// Enable requests logging
//
// If you want to enable requests logging, set the debug flag on your configuration object:
//
//       configuration.Debug = true
//
// Configure proxy
//
// If you want to configure proxy, set env var HTTP_PROXY, and HTTPS_PROXY or set custom
// HTTPClient with proxy configured on configuration object:
//
//       proxyUrl, _ := url.Parse("http://127.0.0.1:80")
//       configuration.HTTPClient = &http.Client{
//           Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
//       }
//
// Pagination
//
// Several listing operations have a pagination method to help consume all the items available.
// For example, to retrieve all your incidents:
//
//
//   package main
//
//   import (
//   	"context"
//   	"fmt"
//   	"os"
//
//   	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
//   	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
//   )
//
//   func main() {
//   	ctx := datadog.NewDefaultContext(context.Background())
//   	configuration := datadog.NewConfiguration()
//   	configuration.SetUnstableOperationEnabled("v2.ListIncidents", true)
//   	apiClient := datadog.NewAPIClient(configuration)
//   	incidentsApi := datadogV2.NewIncidentsApi(apiClient)
//
//   	resp, _ := incidentsApi.ListIncidentsWithPagination(ctx, *datadog.NewListIncidentsOptionalParameters())
//   	for paginationResult := range resp {
//   		if paginationResult.Error != nil {
//   			fmt.Fprintf(os.Stderr, "Error when calling `IncidentsApi.ListIncidentsWithPagination`: %v\n", paginationResult.Error)
//   		}
//   		responseContent, _ := json.MarshalIndent(paginationResult.Item, "", "  ")
//   		fmt.Fprintf(os.Stdout, "%s\n", responseContent)
//   	}
//
//   }
//
// Documentation
//
// Developer documentation for API endpoints and models is available on Github pages (https://datadoghq.dev/datadog-api-client-go/pkg/github.com/DataDog/datadog-api-client-go/v2/).
// Released versions are available on
// pkg.go.dev (https://pkg.go.dev/github.com/DataDog/datadog-api-client-go/v2).
//
// Contributing
//
// As most of the code in this repository is generated, we will only accept PRs for files
// that are not modified by our code-generation machinery (changes to the generated files
// would get overwritten). We happily accept contributions to files that are not autogenerated,
// such as tests and development tooling.
//
//
// Author
//
// support@datadoghq.com
//
//
package client
