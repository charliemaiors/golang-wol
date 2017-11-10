package storage

import (
	"bytes"
	"encoding/gob"
	"errors"

	"github.com/charliemaiors/golang-wol/types"
	storage "github.com/coreos/bbolt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

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

func addDevice(device *types.Device, name string) error {
	log.Debugf("Adding device %v with name %s", device, name)
	buf, err := encodeFromMacIP(device.Mac, device.IP)

	if err != nil {
		log.Errorf("Got error encoding: %v", err)
		return err
	}

	err = db.Update(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(devicesBucket))
		err := bucket.Put([]byte(name), buf.Bytes())
		log.Debugf("Error? %v", err)
		return err
	})
	return err
}

func getAliasesFromStorage(aliasChan chan string) {
	log.Debugf("Got channel %v for alias retrieving", aliasChan)

	db.View(func(transaction *storage.Tx) error {
		cursor := transaction.Bucket([]byte(devicesBucket)).Cursor()
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			log.Debugf("Device %s", string(k))
			aliasChan <- string(k)
		}
		return nil
	})
}

func getDevice(name string) (*types.Device, error) {
	device := &types.Device{}
	log.Debugf("Getting data for device with alias %s", name)

	err := db.View(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(devicesBucket))
		dev := bucket.Get([]byte(name))
		reader := bytes.NewReader(dev)
		err := gob.NewDecoder(reader).Decode(&device)

		if err != nil {
			log.Errorf("Got error decoding: %v", err)
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	log.Debugf("Device is: %v", device)
	return device, nil
}

func checkPassword(pass string) error {

	err := db.View(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(passwordBucket))
		savedPass := bucket.Get([]byte(passworkdKey))
		log.Debugf("Got %s for password from bucket", string(savedPass))
		return bcrypt.CompareHashAndPassword(savedPass, []byte(pass))
	})
	return err
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

func deleteDevice(alias string) error {

	err := db.Update(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(devicesBucket))
		err := bucket.Delete([]byte(alias))
		return err
	})
	return err
}

func updatePassword(oldPassword, newPassword string) error {

	err := db.Update(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(passwordBucket))
		effectiveOldPassHash := bucket.Get([]byte(passworkdKey))
		err := bcrypt.CompareHashAndPassword(effectiveOldPassHash, []byte(oldPassword))
		if err != nil {
			log.Errorf("Got error %v", err)
			return err
		}
		log.Debug("Old password is valid, updating password")
		effectiveNewPasswd, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		log.Debug("Updating password")
		err = bucket.Put([]byte(passworkdKey), effectiveNewPasswd)
		return err
	})
	return err
}

func encodeFromMacIP(mac, IPAddr string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	entry := types.Device{Mac: mac, IP: IPAddr}
	err := gob.NewEncoder(buf).Encode(entry)
	return buf, err
}
