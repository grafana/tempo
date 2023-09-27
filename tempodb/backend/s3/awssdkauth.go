package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	mcreds "github.com/minio/minio-go/v7/pkg/credentials"
)

func NewAWSSDKAuth(region string) *AWSSDKAuth {
	dc := defaults.Config().WithRegion(region)
	creds := defaults.CredChain(dc, defaults.Handlers())
	return &AWSSDKAuth{
		creds: creds,
	}
}

// AWSSDKAuth retrieves credentials from the aws-sdk-go.
type AWSSDKAuth struct {
	creds *credentials.Credentials
}

// Retrieve retrieves the keys from the environment.
func (a *AWSSDKAuth) Retrieve() (mcreds.Value, error) {
	val, err := a.creds.Get()
	if err != nil {
		return mcreds.Value{}, fmt.Errorf("retrieve AWS SDK credentials: %w", err)
	}
	return mcreds.Value{
		AccessKeyID:     val.AccessKeyID,
		SecretAccessKey: val.SecretAccessKey,
		SessionToken:    val.SessionToken,
		SignerType:      mcreds.SignatureV4,
	}, nil
}

// IsExpired returns if the credentials have been retrieved.
func (a *AWSSDKAuth) IsExpired() bool {
	return a.creds.IsExpired()
}
