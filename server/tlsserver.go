package server

import (
	"net/http"
	"github.com/spf13/viper"
)

func StartTLS(){
	err := http.ListenAndServeTLS(:5000,viper.GetString("server.tls.cert"),viper.GetString("server.tls.key"),nil)
	if err != nil {
		panic(err)
	}
}