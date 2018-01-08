package config

import (
	"os"

	"github.com/charliemaiors/golang-wol/server"
	"github.com/charliemaiors/golang-wol/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var srv server.Server

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
	err := configureLog()
	srv = configureSrv()
	if err != nil { //If something went wrong with configured log the application could not start!!!
		panic(err)
	}

	initialized := checkAlreadyRun()
	proxy, proxyPrefix := checkProxy()
	command := getTurnOffCommand()
	port := getTurnOffPort()
	telegram := isBotEnabled() //TODO add to server for telegram instantiation

	log.Debugf("used %s config file", viper.ConfigFileUsed())

	srv.Start(initialized, proxy, telegram, proxyPrefix, command, port)
}

func configureSrv() server.Server {
	if viper.IsSet("server.letsencrypt") {
		log.Debug("configuring letsencrypt")
		return &server.LetsEncryptServer{
			CertDir: viper.GetString("server.letsencrypt.cert"),
			Host:    viper.GetString("server.letsencrypt.host"),
		}
	}
	if viper.IsSet("server.tls") {
		return &server.TLSServer{
			TLSCert: viper.GetString("server.tls.cert"),
			TLSKey:  viper.GetString("server.tls.key"),
		}
	}

	return &server.PlainServer{}
}

func configureLog() error {
	if viper.IsSet("server.log") {
		f, err := os.OpenFile(viper.GetString("server.log"), os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		log.SetOutput(f)
		return nil
	}
	log.SetOutput(os.Stderr)
	return nil
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

func isBotEnabled() bool {
	return viper.IsSet("bot.telegram")
}

func checkProxy() (bool, string) {
	if viper.IsSet("server.proxy") {
		return viper.GetBool("server.proxy.enabled"), viper.GetString("server.proxy.prefix")
	}
	return false, ""
}

func isTelegramEnabled() bool {
	return viper.IsSet("bot.telegram")
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
