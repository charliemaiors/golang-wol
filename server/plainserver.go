package server

import (
	"net/http"
)

func StartNormal() {
	err := http.ListenAndServe(":5000", nil)

	if err != nil {
		panic(err)
	}
}
