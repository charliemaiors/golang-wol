package server

import (
	"net/http"

	"bitbucket.org/cmaiorano/golang-wol/storage"

	"github.com/spf13/viper"
)

//StartTLS deploy the normal tls endpoint secured server
func StartTLS(alreadyInit bool) {
	initialized = alreadyInit
	if initialized {
		go storage.StartHandling(deviceChan, getChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/devices", handleDevices)
	err := http.ListenAndServeTLS(":5000", viper.GetString("server.tls.cert"), viper.GetString("server.tls.key"), nil)
	if err != nil {
		panic(err)
	}

}
