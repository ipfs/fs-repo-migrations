package aws

import "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/aws/aws-sdk-go/aws/awserr"

var (
	// ErrMissingRegion is an error that is returned if region configuration is
	// not found.
	ErrMissingRegion = awserr.New("MissingRegion", "could not find region configuration", nil)

	// ErrMissingEndpoint is an error that is returned if an endpoint cannot be
	// resolved for a service.
	ErrMissingEndpoint = awserr.New("MissingEndpoint", "'Endpoint' configuration is required for this service", nil)
)
