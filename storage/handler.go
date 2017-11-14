package storage

import (
	"github.com/charliemaiors/golang-wol/types"
	storage "github.com/coreos/bbolt"
	log "github.com/sirupsen/logrus"
)

const (
	devicesBucket  = "DevBucket"
	passwordBucket = "PassBucket"
	passworkdKey   = "AdminPassword"
	dbName         = "rwol.db"
	defaultDbLoc   = "storage"
)

var db *storage.DB

func init() {
	log.SetLevel(log.DebugLevel)
}

//StartHandling start an infinite loop in order to handle properly the bbolt database used for alias and password storage
func StartHandling(deviceChan chan *types.AliasResponse, getChan chan *types.GetDev, delDevChan chan *types.DelDev, passHandlingChan chan *types.PasswordHandling, updatePassChan chan *types.PasswordUpdate, getAliases chan chan string) {
	db = getDB()
	defer db.Close()

	for {
		select {
		case newDev := <-deviceChan:
			handleNewDevice(newDev)
		case getDev := <-getChan:
			handleGetDev(getDev)
		case delDev := <-delDevChan:
			handleDeviceDel(delDev)
		case passHandling := <-passHandlingChan:
			handlePass(passHandling)
		case updatePass := <-updatePassChan:
			handleUpdatePass(updatePass)
		case aliasChan := <-getAliases:
			handleAliasRequest(aliasChan)
		}
	}
}

//InitLocal initialize db in case is first start of web application
func InitLocal(initialPassword string) {
	db = getDB()
	log.Debugf("Openend database %v, starting bucket definition", db)

	err := db.Update(func(transaction *storage.Tx) error {
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

	if err != nil {
		panic(err)
	}
}

func handleNewDevice(newDev *types.AliasResponse) {
	log.Debugf("%v", newDev)
	err := addDevice(newDev.Device, newDev.Name)
	if err != nil {
		close(newDev.Response)
	} else {
		newDev.Response <- struct{}{}
		close(newDev.Response)
	}
}

func handleGetDev(getDev *types.GetDev) {
	log.Debug("%v", getDev)
	device, err := getDevice(getDev.Alias)
	if err != nil {
		close(getDev.Response)
	} else {
		getDev.Response <- device
		close(getDev.Response)
	}
}

func handleDeviceDel(delDev *types.DelDev) {
	defer close(delDev.Response)
	err := deleteDevice(delDev.Alias)
	if err != nil {
		delDev.Response <- err
	}
}

func handlePass(passHandling *types.PasswordHandling) {
	defer close(passHandling.Response)
	log.Debugf("%v", passHandling)
	err := checkPassword(passHandling.Password)
	passHandling.Response <- err
}

func handleUpdatePass(updatePass *types.PasswordUpdate) {
	defer close(updatePass.Response)
	log.Debug("%v", updatePass)
	err := updatePassword(updatePass.OldPassword, updatePass.NewPassword)
	updatePass.Response <- err
}

func handleAliasRequest(aliasChan chan string) {
	log.Debug("Got all alias request")
	getAliasesFromStorage(aliasChan)
	close(aliasChan)
}
