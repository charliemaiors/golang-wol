package server

//Server is the interface which configuration process will init the service
type Server interface {
	Start(alreadyInit, reverseProxy, telegram bool, proxyPrefix, command, port string)
}

type ConfigSuccess struct {
	AlreadyInit bool
	Prefix      string
}
