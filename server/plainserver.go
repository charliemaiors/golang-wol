package server

import (
	"net/http"

	"github.com/charliemaiors/golang-wol/storage"
)

//StartNormal start the plain http server without any encryption
func StartNormal(alreadyInit, reverseProxy bool) {
	initialized = alreadyInit
	if initialized {
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}
	loadBox()

	if reverseProxy {
		handleProxy()
	}
	err := http.ListenAndServe(":5000", router)

	if err != nil {
		panic(err)
	}
}
