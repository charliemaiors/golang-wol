package server

import (
	"net/http"

	"github.com/spf13/viper"
)

func StartTLS(alreadyInit bool) {
	initialized = alreadyInit
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/devices", handleDevices)
	err := http.ListenAndServeTLS(":5000", viper.GetString("server.tls.cert"), viper.GetString("server.tls.key"), nil)
	if err != nil {
		panic(err)
	}
}
