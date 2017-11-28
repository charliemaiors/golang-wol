package config

import (
	"os"

	"github.com/charliemaiors/golang-wol/server"
	"github.com/charliemaiors/golang-wol/utils"
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
	proxy := checkProxy()
	command := getTurnOffCommand()
	port := getTurnOffPort()
	log.Debugf("used %s config file", viper.ConfigFileUsed())
	if viper.IsSet("server.letsencrypt") {
		log.Debug("Serving letsencrypt")
		server.StartLetsEncrypt(initialized, proxy, command, port)
	} else if viper.IsSet("server.tls") {
		log.Debug("Serving TLS!")
		server.StartTLS(initialized, proxy, command, port)
	} else {
		log.Debug("Serving Plain!")
		server.StartNormal(initialized, proxy, command, port)
	}
}

func checkAlreadyRun() bool {
	loc := "storage"
	if viper.IsSet("storage.path") {
		loc = viper.GetString("storage.path")
	}
	log.Debugf("Storage location is %s", loc)

	err := utils.CheckIfFolderExist(loc)
	if err != nil { //at least the storage folder could exist or MUST be created
		panic(err)
	}

	if _, err := os.Stat(loc + "/rwol.db"); os.IsNotExist(err) {
		return false
	}
	return true
}

func checkProxy() bool {
	if viper.IsSet("server.proxy") {
		return viper.GetBool("server.proxy")
	}
	return false
}

func getTurnOffCommand() string {
	if viper.IsSet("server.command") {
		return viper.GetString("server.command.option")
	}
	return "poweroff"
}

func getTurnOffPort() string {
	if viper.IsSet("server.command") {
		return viper.GetString("server.command.port")
	}
	return "7740"
}
