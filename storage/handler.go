package storage

import (
	"bytes"
	hash "crypto/sha512"
	"encoding/gob"
	"errors"
	"strings"

	storage "github.com/coreos/bbolt"
	log "github.com/sirupsen/logrus"
	"masinihouse.ddns.net/git/charliemaiors/Golang-Wol/types"
)

const (
	devicesBucket  = "DevBucket"
	passwordBucket = "PassBucket"
	passworkdKey   = "AdminPassword"
)

var db *storage.DB

func init() {
	localDB, err := storage.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Openend database %v, starting bucket definition", localDB)
	db = localDB

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
		log.Errorf("Got erro %v, panic!!!", err)
		panic(err)
	}
}

func StartHandling(deviceChan chan *types.Alias, getChan chan *types.GetDev, passHandlingChan chan *types.PasswordHandling) {

	for {
		select {
		case newDev := <-deviceChan:
			log.Debugf("%v", newDev)
		case getDev := <-getChan:
			log.Debug("%v", getDev)
		}
	}
}

func addDevice(device *types.Device, name string) error {
	log.Debugf("Adding device %v with name %s", device, name)
	buf, err := encodeFromMacIface(device.Mac, device.Iface)

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

func checkPassword(pass string) bool {
	ok := false
	passHash := hash.New()
	effectiveHash := string(passHash.Sum([]byte(pass)))

	db.View(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(passwordBucket))
		savedPass := bucket.Get([]byte(passworkdKey))
		log.Debugf("Got %s for password from bucket", string(savedPass))
		if strings.Compare(string(savedPass), effectiveHash) == 0 {
			ok = true
		}
		return nil
	})
	return ok
}

func insertPassword(pass string) error {
	passHash := hash.New()
	effectiveHash := passHash.Sum([]byte(pass))

	err := db.View(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(passwordBucket))
		if bucket.Get([]byte(passworkdKey)) != nil {
			return errors.New("Password already defined")
		}

		err := bucket.Put([]byte(passworkdKey), effectiveHash)
		log.Errorf("Got error? %v", err)

		return err
	})

	return err
}

func updatePassword(oldPassword, newPassword string) error {
	passHash := hash.New()
	oldPassHash := passHash.Sum([]byte(oldPassword))

	err := db.Update(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(passwordBucket))
		effectiveOldPassHash := bucket.Get([]byte(passworkdKey))
		if bytes.Compare(oldPassHash, effectiveOldPassHash) == 0 {
			passHash.Reset()
			newPassHash := passHash.Sum([]byte(newPassword))
			err := bucket.Put([]byte(passworkdKey), newPassHash)
			return err
		}
		return errors.New("Invalid old password")
	})
	return err
}

func encodeFromMacIface(mac, iface string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	entry := types.Device{Mac: mac, Iface: iface}
	err := gob.NewEncoder(buf).Encode(entry)
	return buf, err
}
