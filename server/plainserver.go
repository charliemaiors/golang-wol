package server

import (
	"net/http"
)

func StartNormal() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/devices", handleDevices)
	err := http.ListenAndServe(":5000", nil)

	if err != nil {
		panic(err)
	}
}
