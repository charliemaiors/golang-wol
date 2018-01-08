package server

import (
	"crypto/tls"
	"net/http"

	"github.com/charliemaiors/golang-wol/bot"
	"github.com/charliemaiors/golang-wol/storage"
	"github.com/charliemaiors/golang-wol/utils"

	"golang.org/x/crypto/acme/autocert"
)

type LetsEncryptServer struct {
	Host    string
	CertDir string
}

//StartLetsEncrypt spawn a https web server powered by letsencrypt certificates
func (srv *LetsEncryptServer) Start(alreadyInit, reverseProxy, telegram bool, proxyPrefix, command, port string) {
	initialized = alreadyInit
	prefix = proxyPrefix

	err := utils.CheckIfFolderExist(srv.CertDir)
	if err != nil { //Please insert a valid cert path
		panic(err)
	}

	if initialized {
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}

	if telegram { //telegram bot does not require any password because of the authorized user
		go bot.RunBot(deviceChan, getChan, delDevChan, aliasRequestChan)
	}

	if reverseProxy {
		handleProxy()
	}
	solcommand = command
	turnOffPort = port

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(srv.Host), //your domain here
		Cache:      autocert.DirCache(srv.CertDir),   //folder for storing certificates
	}

	configureBox()

	server := &http.Server{
		Addr: ":443", //Different port from 443 could be hard for acme-tlsni-challenge
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
		Handler: handler,
	}

	err = server.ListenAndServeTLS("", "")

	if err != nil {
		panic(err)
	}
}
