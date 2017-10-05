package server

import (
	"errors"
	"net/http"

	"bitbucket.org/cmaiorano/golang-wol/types"
	wol "github.com/sabhiram/go-wol"
)

var initialized = false
var deviceChan = make(chan *types.Alias)
var getChan = make(chan *types.GetDev)
var passHandlingChan = make(chan *types.PasswordHandling)
var updatePassChan = make(chan *types.PasswordUpdate)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if !initialized {
		redirectToConfig(w, r)
		return
	}
}

func handleDevices(w http.ResponseWriter, r *http.Request) {
	if !initialised {
		redirectToConfig(w, r)
		return
	}
}

func redirectToConfig(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/config", 301)
}

func registerDevice(alias, mac, iface string) error {
	dev := &types.Device{Iface: iface, Mac: mac}
	resp := make(chan struct{}, 1)
	aliasStr := &types.Alias{Device: dev, Name: alias, Response: resp}
	deviceChan <- aliasStr

	if value, ok := <-resp; !ok {
		return errors.New("Error adding device")
	}
	return nil
}

func sendPacket(computerName, localIface string) error {
	macAddr, bcastAddr = "", ""
	err := wol.SendMagicPacket(macAddr, bcastAddr, localIface)
	return err
}
