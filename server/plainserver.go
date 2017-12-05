package server

import (
	"net/http"

	"github.com/charliemaiors/golang-wol/bot"
	"github.com/charliemaiors/golang-wol/storage"
)

//StartNormal start the plain http server without any encryption
func StartNormal(alreadyInit, reverseProxy, telegram bool, command, port string) {
	initialized = alreadyInit

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
	err := http.ListenAndServe(":5000", handler)

	if err != nil {
		panic(err)
	}
}
