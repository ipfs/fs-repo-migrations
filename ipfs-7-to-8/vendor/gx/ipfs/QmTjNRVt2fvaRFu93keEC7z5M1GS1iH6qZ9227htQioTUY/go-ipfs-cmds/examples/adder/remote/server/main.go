package main

import (
	nethttp "net/http"

	"github.com/ipfs/go-ipfs-cmds/examples/adder"

	http "github.com/ipfs/go-ipfs-cmds/http"
)

func main() {
	h := http.NewHandler(nil, adder.RootCmd, http.NewServerConfig())

	// create http rpc server
	err := nethttp.ListenAndServe(":6798", h)
	if err != nil {
		panic(err)
	}
}
