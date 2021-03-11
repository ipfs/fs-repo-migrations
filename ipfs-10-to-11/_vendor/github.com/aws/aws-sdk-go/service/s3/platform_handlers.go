// +build !go1.6

package s3

import "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/aws/aws-sdk-go/aws/request"

func platformRequestHandlers(r *request.Request) {
}
