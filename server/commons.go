package server

import (
	wol "github.com/sabhiram/go-wol"
)

func registerDevice()

func sendPacket(computerName, localIface string) error {
	macAddr, bcastAddr = "", ""
	err := wol.SendMagicPacket(macAddr, bcastAddr, localIface)
	return err
}
