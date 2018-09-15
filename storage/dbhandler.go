package storage

import (
	storage "github.com/coreos/bbolt"
)

func initHandler(storagePath string, dbchan chan func(*storage.DB)) {
	if storagePath == "" {
		panic("Storage path is empty")
	}

	db, err := storage.Open(storagePath, 0600, nil)

	if err != nil {
		panic(err)
	}

	for {
		dbReq := <-dbchan
		dbReq(db)
	}
}
