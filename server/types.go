package server

import "github.com/charliemaiors/golang-wol/types"

//Server is the interface which configuration process will init the service
type Server interface {
	Start(alreadyInit, reverseProxy, telegram bool, proxyPrefix, command, port string)
}

type ConfigSuccess struct {
	AlreadyInit bool
	Prefix      string
}

type DeviceListRevProxy struct {
	Devices map[string]*types.Device
	Prefix  string
}
