package server

import (
	"net/http"

	"github.com/charliemaiors/golang-wol/bot"
	"github.com/charliemaiors/golang-wol/storage"
)

//TLSServer is the structure used in order to deploy a TLS secured server
type TLSServer struct {
	TLSCert string
	TLSKey  string
}

//Start deploy the normal tls endpoint secured server
func (srv *TLSServer) Start(alreadyInit, reverseProxy, telegram bool, proxyPrefix, command, port string) {
	initialized = alreadyInit
	prefix = proxyPrefix

	if initialized {
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}

	if telegram { //telegram bot does not require any password because of the authorized user
		go bot.RunBot(deviceChan, getChan, delDevChan, aliasRequestChan)
	}

	configureBox()
	if reverseProxy {
		handleProxy()
	}
	solcommand = command
	turnOffPort = port

	err := http.ListenAndServeTLS(":5000", srv.TLSCert, srv.TLSKey, handler)
	if err != nil {
		panic(err)
	}

}
