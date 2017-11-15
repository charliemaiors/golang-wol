package server

import (
	"crypto/tls"
	"net/http"

	"github.com/charliemaiors/golang-wol/storage"

	"github.com/spf13/viper"

	"golang.org/x/crypto/acme/autocert"
)

//StartLetsEncrypt spawn a https web server powered by letsencrypt certificates
func StartLetsEncrypt(alreadyInit, reverseProxy bool, command, port string) {
	initialized = alreadyInit
	host := viper.GetString("server.letsencrypt.host")
	certDir := viper.GetString("server.letsencrypt.cert")

	err := checkIfFolderExist(certDir)
	if err != nil { //Please insert a valid cert path
		panic(err)
	}

	if initialized {
		go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)
	}

	if reverseProxy {
		handleProxy()
	}
	solcommand = command
	turnOffPort = port

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(host), //your domain here
		Cache:      autocert.DirCache(certDir),   //folder for storing certificates
	}

	server := &http.Server{
		Addr: ":443", //Different port from 443 could be hard for acme-tlsni-challenge
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
		Handler: handler,
	}
	loadBox()

	err = server.ListenAndServeTLS("", "")

	if err != nil {
		panic(err)
	}
}
