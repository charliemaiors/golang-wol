package storage

import (
	"bytes"
	hash "crypto/sha512"
	"encoding/gob"
	"errors"
	"os"
	"strings"

	"bitbucket.org/cmaiorano/golang-wol/types"
	storage "github.com/coreos/bbolt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	devicesBucket  = "DevBucket"
	passwordBucket = "PassBucket"
	passworkdKey   = "AdminPassword"
	dbName         = "rwol.db"
	defaultDbLoc   = "storage"
)

var db *storage.DB

//StartHandling start an infinite loop in order to handle properly the bbolt database used for alias and password storage
func StartHandling(initialPassword string, deviceChan chan *types.Alias, getChan chan *types.GetDev, passHandlingChan chan *types.PasswordHandling, updatePassChan chan *types.PasswordUpdate, getAliases chan chan string) {

	initLocal()

	err := insertPassword(initialPassword)
	if err != nil {
		panic(err)
		os.Exit(2)
	}

	for {
		select {
		case newDev := <-deviceChan:
			log.Debugf("%v", newDev)
			err := addDevice(newDev.Device, newDev.Name)
			if err != nil {
				close(newDev.Response)
			} else {
				newDev.Response <- struct{}{}
				close(newDev.Response)
			}
		case getDev := <-getChan:
			log.Debug("%v", getDev)
			device, err := getDevice(getDev.Alias)
			if err != nil {
				close(getDev.Response)
			} else {
				getDev.Response <- device
				close(getDev.Response)
			}
		case passHandling := <-passHandlingChan:
			log.Debug("%v", passHandling)
			err := checkPassword(passHandling.Password)
			passHandling.Response <- err
			close(passHandling.Response)

		case updatePass := <-updatePassChan:
			log.Debug("%v", updatePass)
			err := updatePassword(updatePass.OldPassword, updatePass.NewPassword)
			updatePass.Response <- err
			close(updatePass.Response)

		case aliasChan := <-getAliases:
			log.Debug("Got all alias request")
			getAliasesFromStorage(aliasChan)
			close(aliasChan)
		}
	}
}

func initLocal() {
	dbLoc := defaultDbLoc
	if viper.IsSet("storage.path") {
		dbLoc = viper.GetString("storage.path")
	}

	localDB, err := storage.Open(dbLoc+"/"+dbName, 0600, nil)
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
		log.Errorf("Got err %v, panic!!!", err)
		panic(err)
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

func getAliasesFromStorage(aliasChan chan string) {
	log.Debugf("Got channel %v for alias retrieving", aliasChan)

	db.View(func(transaction *storage.Tx) error {
		cursor := transaction.Bucket([]byte(devicesBucket)).Cursor()
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
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
	passHash := hash.New()
	effectiveHash := string(passHash.Sum([]byte(pass)))

	err := db.View(func(transaction *storage.Tx) error {
		bucket := transaction.Bucket([]byte(passwordBucket))
		savedPass := bucket.Get([]byte(passworkdKey))
		log.Debugf("Got %s for password from bucket", string(savedPass))
		if strings.Compare(string(savedPass), effectiveHash) == 0 {
			return errors.New("Different Password")
		}
		return nil
	})
	return err
}

func insertPassword(pass string) error {
	passHash := hash.New()
	effectiveHash := passHash.Sum([]byte(pass))

	err := db.Update(func(transaction *storage.Tx) error {
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
