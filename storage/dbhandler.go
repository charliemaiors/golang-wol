package storage

import (
	storage "github.com/coreos/bbolt"
	log "github.com/sirupsen/logrus"
)

func initHandler(storagePath, initialPassword string, dbchan chan func(*storage.DB)) {
	if storagePath == "" {
		panic("Storage path is empty")
	}

	db, err := storage.Open(storagePath, 0600, nil)

	if err != nil {
		panic(err)
	}

	err = db.Update(func(transaction *storage.Tx) error {
		if _, createErr := transaction.CreateBucketIfNotExists([]byte(devicesBucket)); createErr != nil {
			log.Errorf("Error creating devicesBucket: %v", createErr)
			return createErr
		}
		if _, createErr := transaction.CreateBucketIfNotExists([]byte(passwordBucket)); createErr != nil {
			log.Errorf("Error creating passwordBucket: %v", createErr)
			return createErr
		}
		return nil
	})

	if err != nil {
		log.Errorf("Got err %v, panic!!!", err)
		panic(err)
	}

	err = insertPassword(initialPassword)

	for {
		dbReq := <-dbchan
		dbReq(db)
	}
}
