// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package msk implements the required IAM auth used by AWS' managed Kafka platform
// to be used with the Surama kafka producer.
//
// Further details on how the SASL connector works can be viewed here:
//
//	https://github.com/aws/aws-msk-iam-auth#details
package awsmsk // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter/internal/awsmsk"
