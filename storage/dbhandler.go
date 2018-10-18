package storage

import (
	"bytes"
	"encoding/gob"
	"errors"

	"github.com/charliemaiors/golang-wol/types"
	storage "github.com/coreos/bbolt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func InitHandler(storagePath, initialPassword string, dbchan chan func(*storage.DB)) (BboltDeviceAccessor, BboltPasswordAccessor, BboltSchedulerAccessor) {
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
		if _, createErr := transaction.CreateBucketIfNotExists([]byte(scheduleBucket)); createErr != nil {
			log.Errorf("Error creating scheduleBucket: %v", createErr)
			return createErr
		}
		return nil
	})

	if err != nil {
		log.Errorf("Got err %v, panic!!!", err)
		panic(err)
	}

	err = insertPassword(initialPassword)

	go func() {
		for {
			dbReq := <-dbchan
			dbReq(db)
		}
	}()

	return newBboltDeviceAccessor(dbchan), newBboltPasswordAccessor(dbchan), newBboltSchedulerAccessor(dbchan)
}

func insertPassword(pass string) error {
	passHash := []byte(pass)
	effectivePasswd, err := bcrypt.GenerateFromPassword(passHash, bcrypt.DefaultCost)

	if err != nil {
		panic(err)
	}

	log.Debugf("Generated password %v", effectivePasswd)

	err = db.Update(func(transaction *storage.Tx) error {
		log.Debug("Entering transaction")
		bucket := transaction.Bucket([]byte(passwordBucket))
		if bucket.Get([]byte(passworkdKey)) != nil {
			return errors.New("Password already defined")
		}

		log.Debug("Adding password")
		err := bucket.Put([]byte(passworkdKey), effectivePasswd)
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

func encodeFromMacIP(mac, IPAddr string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	entry := types.Device{Mac: mac, IP: IPAddr}
	err := gob.NewEncoder(buf).Encode(entry)
	return buf, err
}

func encodeActionSubj(alias string, action types.Action) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	subj := ActionSubject{
		Action: action,
		Alias:  alias,
	}
	err := gob.NewEncoder(buf).Encode(subj)
	return buf, err
}
