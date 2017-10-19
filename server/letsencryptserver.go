package server

import (
	"crypto/tls"
	"net/http"

	"github.com/spf13/viper"

	"golang.org/x/crypto/acme/autocert"
)

//StartLetsEncrypt spawn a https web server powered by letsencrypt certificates
func StartLetsEncrypt(alreadyInit bool) {

	host := viper.GetString("server.letsencrypt.host")
	certDir := viper.GetString("server.letsencrypt.cert")

	err := checkIfFolderExist(certDir)
	if err != nil { //Please insert a valid cert path
		panic(err)
	}

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(host), //your domain here
		Cache:      autocert.DirCache(certDir),   //folder for storing certificates
	}

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/devices", handleDevices)

	server := &http.Server{
		Addr: ":443", //Different port from 443 could be hard for challenges
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	err = server.ListenAndServeTLS("", "")
	if err != nil {
		panic(err)
	}
}