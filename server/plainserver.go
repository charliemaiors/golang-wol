package server

import (
	"net/http"

	"github.com/charliemaiors/golang-wol/storage"
)

//StartNormal start the plain http server without any encryption
func StartNormal(alreadyInit, reverseProxy bool, command string) {
	initialized = alreadyInit
	if initialized {
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}
	loadBox()

	if reverseProxy {
		handleProxy()
	}

	solcommand = command
	err := http.ListenAndServe(":5000", handler)

	if err != nil {
		panic(err)
	}
}
