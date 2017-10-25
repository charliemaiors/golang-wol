package server

import (
	"net/http"

	"bitbucket.org/cmaiorano/golang-wol/storage"
)

//StartNormal start the plain http server without any encryption
func StartNormal(alreadyInit bool) {
	initialized = alreadyInit
	if initialized {
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}
	loadBox()
	err := http.ListenAndServe(":5000", router)

	if err != nil {
		panic(err)
	}
}
