package server

import (
	"net/http"

	"github.com/charliemaiors/golang-wol/bot"
	"github.com/charliemaiors/golang-wol/storage"

	"github.com/spf13/viper"
)

//StartTLS deploy the normal tls endpoint secured server
func StartTLS(alreadyInit, reverseProxy, telegram bool, command, port string) {
	initialized = alreadyInit
	telebot = telegram

	if initialized {
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}

	if telegram { //telegram bot does not require any password because of the authorized user
		go bot.RunBot(deviceChan, getChan, delDevChan, aliasRequestChan)
	}

	loadBox()
	if reverseProxy {
		handleProxy()
	}
	solcommand = command
	turnOffPort = port

	err := http.ListenAndServeTLS(":5000", viper.GetString("server.tls.cert"), viper.GetString("server.tls.key"), handler)
	if err != nil {
		panic(err)
	}

}
