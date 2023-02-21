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

package k8s // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metadata/internal/k8s"

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig"
)

type nodeNameProvider interface {
	NodeName(context.Context) (string, error)
}

var _ nodeNameProvider = (*nodeNameProviderImpl)(nil)

type nodeNameProviderImpl struct {
	logger *zap.Logger
	client k8s.Interface
}

func (p *nodeNameProviderImpl) namespace() string {
	namespacePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	val, err := os.ReadFile(namespacePath)
	if err == nil && val != nil {
		return string(val)
	}
	p.logger.Warn("Could not fetch k8s namespace, using 'default'", zap.Error(err))
	return "default"
}

func (p *nodeNameProviderImpl) NodeName(ctx context.Context) (string, error) {
	namespace := p.namespace()

	// NOTE: The pod name may not match the OS hostname, e.g. if it has been modified
	// via the 'setHostnameAsFQDN' and 'hostname' fields in the pod spec.
	// The query below will error out in that case. See:
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/11033
	podName, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("could not fetch pod hostname: %w", err)
	}

	// NOTE: If changing this, check if the RBAC rules on the docs or examples need updates.
	pod, err := p.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return pod.Spec.NodeName, nil
}

var _ nodeNameProvider = (*nodeNameUnavailable)(nil)

type nodeNameUnavailable struct {
	err error
}

func (n *nodeNameUnavailable) NodeName(context.Context) (string, error) {
	return "", fmt.Errorf("k8s client is unavailable: %w", n.err)
}

func newNodeNameProvider() nodeNameProvider {
	client, err := k8sconfig.MakeClient(k8sconfig.APIConfig{
		AuthType: k8sconfig.AuthTypeServiceAccount,
	})

	if err != nil {
		return &nodeNameUnavailable{err: err}
	}

	return &nodeNameProviderImpl{
		client: client,
	}
}
