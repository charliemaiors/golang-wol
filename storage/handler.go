package storage

import (
	"log"

	storage "github.com/coreos/bbolt"
)

var db *storage.DB

func init() {
	localDB, err := storage.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	db = localDB
}

func StartHandling()
