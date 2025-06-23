// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"

	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka/awsmsk"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka"
)

// configureSaramaAuthentication configures authentication in sarama.Config.
//
// The provided config is assumed to have been validated.
func configureSaramaAuthentication(
	ctx context.Context,
	config configkafka.AuthenticationConfig,
	saramaConfig *sarama.Config,
) {
	if config.PlainText != nil {
		configurePlaintext(*config.PlainText, saramaConfig)
	}
	if config.SASL != nil {
		configureSASL(ctx, *config.SASL, saramaConfig)
	}
	if config.Kerberos != nil {
		configureKerberos(*config.Kerberos, saramaConfig)
	}
}

func configurePlaintext(config configkafka.PlainTextConfig, saramaConfig *sarama.Config) {
	saramaConfig.Net.SASL.Enable = true
	saramaConfig.Net.SASL.User = config.Username
	saramaConfig.Net.SASL.Password = config.Password
}

func configureSASL(ctx context.Context, config configkafka.SASLConfig, saramaConfig *sarama.Config) {
	saramaConfig.Net.SASL.Enable = true
	saramaConfig.Net.SASL.User = config.Username
	saramaConfig.Net.SASL.Password = config.Password
	saramaConfig.Net.SASL.Version = int16(config.Version)

	switch config.Mechanism {
	case SCRAMSHA512:
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: sha512.New} }
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	case SCRAMSHA256:
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: sha256.New} }
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
	case PLAIN:
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	case AWSMSKIAM:
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return awsmsk.NewIAMSASLClient(config.AWSMSK.BrokerAddr, config.AWSMSK.Region, saramaConfig.ClientID)
		}
		saramaConfig.Net.SASL.Mechanism = awsmsk.Mechanism
	case AWSMSKIAMOAUTHBEARER:
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		saramaConfig.Net.SASL.TokenProvider = &awsMSKTokenProvider{ctx: ctx, region: config.AWSMSK.Region}
	}
}

func configureKerberos(config configkafka.KerberosConfig, saramaConfig *sarama.Config) {
	saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
	saramaConfig.Net.SASL.Enable = true
	if config.UseKeyTab {
		saramaConfig.Net.SASL.GSSAPI.KeyTabPath = config.KeyTabPath
		saramaConfig.Net.SASL.GSSAPI.AuthType = sarama.KRB5_KEYTAB_AUTH
	} else {
		saramaConfig.Net.SASL.GSSAPI.AuthType = sarama.KRB5_USER_AUTH
		saramaConfig.Net.SASL.GSSAPI.Password = config.Password
	}
	saramaConfig.Net.SASL.GSSAPI.KerberosConfigPath = config.ConfigPath
	saramaConfig.Net.SASL.GSSAPI.Username = config.Username
	saramaConfig.Net.SASL.GSSAPI.Realm = config.Realm
	saramaConfig.Net.SASL.GSSAPI.ServiceName = config.ServiceName
	saramaConfig.Net.SASL.GSSAPI.DisablePAFXFAST = config.DisablePAFXFAST
}

type awsMSKTokenProvider struct {
	ctx    context.Context
	region string
}

// Token return the AWS session token for the AWS_MSK_IAM_OAUTHBEARER mechanism
func (c *awsMSKTokenProvider) Token() (*sarama.AccessToken, error) {
	token, _, err := signer.GenerateAuthToken(c.ctx, c.region)
	return &sarama.AccessToken{Token: token}, err
}
