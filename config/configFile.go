package config

import (
	"os"

	"bitbucket.org/cmaiorano/golang-wol/server"
	"bitbucket.org/cmaiorano/golang-wol/storage"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	viper.AddConfigPath("./config/")
	viper.AddConfigPath("/etc/wol/")
	viper.SetConfigName("wol")
	log.SetLevel(log.DebugLevel)
}

func checkAlreadyRun() bool {
	loc := "storage"
	if viper.IsSet("storage.path") {
		loc = viper.GetString("storage.path")
	}
	log.Debugf("Storage location is %s", loc)

	if _, err := os.Stat(loc + "/rwol.db"); os.IsNotExist(err) {
		return false
	}
	return true
}

//Start is used to start the service with provided configuration
func Start() {
	initialized := checkAlreadyRun()
	storage.InitLocal()
	if viper.IsSet("server.tls") {
		log.Debug("Serving TLS!")
		server.StartTLS(initialized)
	} else {
		log.Debug("Serving Plain!")
		server.StartNormal(initialized)
	}
}
