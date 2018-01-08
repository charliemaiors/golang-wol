package server

import (
	"net/http"

	"github.com/charliemaiors/golang-wol/bot"
	"github.com/charliemaiors/golang-wol/storage"
)

//PlainServer represent the structure for http plain server
type PlainServer struct {
}

//Start start the plain http server without any encryption
func (srv *PlainServer) Start(alreadyInit, reverseProxy, telegram bool, proxyPrefix, command, port string) {
	initialized = alreadyInit
	prefix = proxyPrefix

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
