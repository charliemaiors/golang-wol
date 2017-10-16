package config

import (
	"bitbucket.org/cmaiorano/golang-wol/server"
	"github.com/spf13/viper"
)

func init() {
	viper.AddConfigPath("./config/")
	viper.AddConfigPath("/etc/wol/")
	viper.SetConfigName("wol")
}

//Start is used to start the service with provided configuration
func Start() {
	if viper.IsSet("server.tls") {
		server.StartTLS()
	} else {
		server.StartNormal()
	}
}
