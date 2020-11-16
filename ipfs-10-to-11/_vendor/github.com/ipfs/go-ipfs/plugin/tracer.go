package plugin

import (
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/opentracing/opentracing-go"
)

// PluginTracer is an interface that can be implemented to add a tracer
type PluginTracer interface {
	Plugin

	InitTracer() (opentracing.Tracer, error)
}
