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
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}
	loadBox()
	err := http.ListenAndServeTLS(":5000", viper.GetString("server.tls.cert"), viper.GetString("server.tls.key"), router)
	if err != nil {
		panic(err)
	}

}
