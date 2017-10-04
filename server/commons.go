package server

import (
	"errors"

	"bitbucket.org/cmaiorano/golang-wol/types"
	wol "github.com/sabhiram/go-wol"
)

var deviceChan = make(chan *types.Alias)
var getChan = make(chan *types.GetDev)
var passHandlingChan = make(chan *types.PasswordHandling)
var updatePassChan = make(chan *types.PasswordUpdate)

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
