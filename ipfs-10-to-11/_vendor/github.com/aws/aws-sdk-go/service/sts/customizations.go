package sts

import "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/aws/aws-sdk-go/aws/request"

func init() {
	initRequest = customizeRequest
}

func customizeRequest(r *request.Request) {
	r.RetryErrorCodes = append(r.RetryErrorCodes, ErrCodeIDPCommunicationErrorException)
}
