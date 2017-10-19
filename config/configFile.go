package config

import (
	"errors"
	"os"

	"bitbucket.org/cmaiorano/golang-wol/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	viper.AddConfigPath("./config/")
	viper.AddConfigPath("/etc/wol/")
	viper.SetConfigName("wol")
	err := viper.ReadInConfig()
	if err != nil {
		log.Errorf("No config file readed: %v", err)
	}
	log.SetLevel(log.DebugLevel)
}

//Start is used to start the service with provided configuration
func Start() {
	initialized := checkAlreadyRun()
	log.Debugf("used %s config file", viper.ConfigFileUsed())
	if viper.IsSet("server.letsencrypt") {
		log.Debug("Serving letsencrypt")
		server.StartLetsEncrypt(initialized)
	} else if viper.IsSet("server.tls") {
		log.Debug("Serving TLS!")
		server.StartTLS(initialized)
	} else {
		log.Debug("Serving Plain!")
		server.StartNormal(initialized)
	}
}

func checkAlreadyRun() bool {
	loc := "storage"
	if viper.IsSet("storage.path") {
		loc = viper.GetString("storage.path")
	}
	log.Debugf("Storage location is %s", loc)

	err := checkIfFolderExist(loc)
	if err != nil { //at least the storage folder could exist or MUST be created
		panic(err)
	}

	if _, err := os.Stat(loc + "/rwol.db"); os.IsNotExist(err) {
		return false
	}
	return true
}

func checkIfFolderExist(loc string) error {
	info, err := os.Stat(loc)
	if os.IsNotExist(err) {
		err = os.MkdirAll(loc, os.ModeDir)
		return err
	} else if !info.IsDir() {
		return errors.New("Exist but is not a folder")
	}
	return nil
}
