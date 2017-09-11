// +build !test

package aws

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

type sessionParameters struct {
	accessKey  string
	secretKey  string
	region     string
	iamRole    string
	iamSession string
}

func newAwsSession(parameters *sessionParameters) (*session.Session, error) {
	if parameters.region == "" {
		return nil, errors.New("Missing aws region (required).")
	}

	sess := session.New(&aws.Config{Region: aws.String(parameters.region)})

	if parameters.accessKey != "" && parameters.secretKey != "" {
		sess = session.New(&aws.Config{
			Region:      aws.String(parameters.region),
			Credentials: credentials.NewStaticCredentials(parameters.accessKey, parameters.secretKey, ""),
		})
	}

	if parameters.iamRole != "" {
		creds := assumeRoleCredentials(sess, parameters.iamRole, parameters.iamSession)
		sess.Config.Credentials = creds
	}

	return sess, nil
}

func assumeRoleCredentials(sess *session.Session, iamRole, iamSession string) *credentials.Credentials {

	if iamSession == "" {
		iamSession = "default"
	}

	creds := stscreds.NewCredentials(sess, iamRole, func(o *stscreds.AssumeRoleProvider) {
		o.Duration = time.Hour
		o.ExpiryWindow = 5 * time.Minute
		o.RoleSessionName = iamSession
	})
	return creds
}
