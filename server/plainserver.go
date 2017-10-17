package server

import (
	"net/http"

	"bitbucket.org/cmaiorano/golang-wol/storage"
)

//StartNormal start the plain http server without any encryption
func StartNormal(alreadyInit bool) {
	initialized = alreadyInit
	if initialized {
		go storage.StartHandling(deviceChan, getChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/devices", handleDevices)
	err := http.ListenAndServe(":5000", nil)

	if err != nil {
		panic(err)
	}
}
