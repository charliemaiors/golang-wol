package server

import (
	"net/http"

	"github.com/charliemaiors/golang-wol/storage"

	"github.com/spf13/viper"
)

//StartTLS deploy the normal tls endpoint secured server
func StartTLS(alreadyInit, reverseProxy bool, command string) {
	initialized = alreadyInit

	if initialized {
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}

	loadBox()
	if reverseProxy {
		handleProxy()
	}
	solcommand = command

	err := http.ListenAndServeTLS(":5000", viper.GetString("server.tls.cert"), viper.GetString("server.tls.key"), handler)
	if err != nil {
		panic(err)
	}

}
