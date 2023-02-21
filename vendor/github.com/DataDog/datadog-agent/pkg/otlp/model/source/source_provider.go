// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package source

import (
	"context"
	"fmt"
)

// Kind of source
type Kind string

const (
	// InvalidKind is an invalid kind. It is the zero value of Kind.
	InvalidKind Kind = ""
	// HostnameKind is a host source.
	HostnameKind Kind = "host"
	// AWSECSFargateKind is a serverless source on AWS ECS Fargate.
	AWSECSFargateKind Kind = "task_arn"
)

// Source represents a telemetry source.
type Source struct {
	// Kind of source (serverless v. host).
	Kind Kind
	// Identifier that uniquely determines the source.
	Identifier string
}

// Tag associated to a source.
func (s *Source) Tag() string {
	return fmt.Sprintf("%s:%s", s.Kind, s.Identifier)
}

// Provider identifies a source.
type Provider interface {
	// Source gets the source from the current context.
	Source(ctx context.Context) (Source, error)
}
