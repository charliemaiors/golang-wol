package storage

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/charliemaiors/golang-wol/types"
	storage "github.com/coreos/bbolt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

var db *storage.DB

func getDB() *storage.DB {

	if db == nil {
		dbLoc := defaultDbLoc
		if viper.IsSet("storage.path") {
			dbLoc = viper.GetString("storage.path")
		}

		localDB, err := storage.Open(dbLoc+"/"+dbName, 0600, nil)
		if err != nil {
			panic(err)
		}
		return localDB
	}
	return db
}

func (accessor *BboltDeviceAccessor) AddDevice(device *types.Device, name string) error {
	log.Debugf("Adding device %v with name %s", device, name)
	buf, err := encodeFromMacIP(device.Mac, device.IP)

	if err != nil {
		log.Errorf("Got error encoding: %v", err)
		return err
	}

	responseChan := make(chan error)

	add := func(db *storage.DB) {
		err = db.Update(func(transaction *storage.Tx) error {
			bucket := transaction.Bucket([]byte(devicesBucket))
			err := bucket.Put([]byte(name), buf.Bytes())
			log.Debugf("Error? %v", err)
			return err
		})

		if err != nil {
			responseChan <- err
		}
		close(responseChan)
	}

	accessor.reqdb <- add
	return <-responseChan
}

func (accessor *BboltDeviceAccessor) GetDevice(alias string) (device *types.Device, err error) {

	log.Debugf("Getting data for device with alias %s", alias)
	errChan := make(chan error)

	get := func(db *storage.DB) {
		err := db.View(func(transaction *storage.Tx) error {
			bucket := transaction.Bucket([]byte(devicesBucket))
			dev := bucket.Get([]byte(alias))
			reader := bytes.NewReader(dev)
			err := gob.NewDecoder(reader).Decode(&device)

			return err
		})
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}

	accessor.reqdb <- get
	if err = <-errChan; err != nil {
		return nil, err
	}
	log.Debugf("Device is: %v", device)
	return device, nil

}

func (accessor *BboltDeviceAccessor) GetAllAliases() []string {

	aliasChan := make(chan string)
	allAliases := make([]string, 0, 0)

	aliases := func(db *storage.DB) {
		db.View(func(transaction *storage.Tx) error {
			cursor := transaction.Bucket([]byte(devicesBucket)).Cursor()
			for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
				log.Debugf("Device %s", string(k))
				aliasChan <- string(k)
			}
			return nil
		})
	}

	accessor.reqdb <- aliases

	for alias := range aliasChan {
		allAliases = append(allAliases, alias)
	}
	return allAliases
}

func (accessor *BboltDeviceAccessor) DeleteDevice(alias string) error {
	errChan := make(chan error)

	delete := func(db *storage.DB) {
		err := db.Update(func(transaction *storage.Tx) error {
			bucket := transaction.Bucket([]byte(devicesBucket))
			err := bucket.Delete([]byte(alias))
			return err
		})
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}

	accessor.reqdb <- delete
	return <-errChan
}

func (accessor *BboltPasswordAccessor) CheckPassword(password string) error {
	errChan := make(chan error)
	check := func(db *storage.DB) {
		err := db.View(func(transaction *storage.Tx) error {
			bucket := transaction.Bucket([]byte(passwordBucket))
			savedPass := bucket.Get([]byte(passworkdKey))
			log.Debugf("Got %s for password from bucket", string(savedPass))
			return bcrypt.CompareHashAndPassword(savedPass, []byte(password))
		})
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}
	accessor.reqdb <- check
	return <-errChan
}

func (accessor *BboltPasswordAccessor) UpdatePassword(oldpass, newpass string) error {

	errChan := make(chan error)
	update := func(db *storage.DB) {
		err := db.Update(func(transaction *storage.Tx) error {
			bucket := transaction.Bucket([]byte(passwordBucket))
			effectiveOldPassHash := bucket.Get([]byte(passworkdKey))
			err := bcrypt.CompareHashAndPassword(effectiveOldPassHash, []byte(oldpass))
			if err != nil {
				log.Errorf("Got error %v", err)
				return err
			}
			log.Debug("Old password is valid, updating password")
			effectiveNewPasswd, err := bcrypt.GenerateFromPassword([]byte(newpass), bcrypt.DefaultCost)
			log.Debug("Updating password")
			err = bucket.Put([]byte(passworkdKey), effectiveNewPasswd)
			return err
		})
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}
	accessor.reqdb <- update
	return <-errChan
}

func (accessor *BboltSchedulerAccessor) ScheduleOperation(alias string, action types.Action, when time.Time) error {
	errChan := make(chan error)
	buffer, err := encodeActionSubj(alias, action)

	if err != nil {
		log.Errorf("Got error %v", err)
		return err
	}

	schedule := func(db *storage.DB) {
		err := db.Update(func(transaction *storage.Tx) error {
			bucket := transaction.Bucket([]byte(scheduleBucket))
			err := bucket.Put(buffer.Bytes(), []byte(when.String()))
			return err
		})
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}
	accessor.reqdb <- schedule

	return <-errChan
}
