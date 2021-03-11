// +build !go1.7

package v4

import (
	"net/http"

	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/aws/aws-sdk-go/aws"
)

func requestContext(r *http.Request) aws.Context {
	return aws.BackgroundContext()
}
