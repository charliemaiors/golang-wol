package server

import (
	"errors"

	"github.com/charliemaiors/golang-wol/types"
	log "github.com/sirupsen/logrus"
)

func delDevice(alias string) error {
	resp := make(chan error)
	delDev := &types.DelDev{
		Alias:    alias,
		Response: resp,
	}
	log.Debugf("Sending delete request with %v", delDev)
	delDevChan <- delDev
	err, ok := <-resp
	if ok && err != nil {
		return err
	}
	return nil
}

func checkPassword(password string) error {
	respChan := make(chan error)
	pass := &types.PasswordHandling{Password: password, Response: respChan}
	passHandlingChan <- pass
	err := <-respChan
	return err
}

func registerOrUpdateDevice(alias, mac, ip string) (*types.Alias, error) {
	if !reMAC.MatchString(mac) {
		return nil, errors.New("Invalid mac address format")
	}

	dev := &types.Device{Mac: mac, IP: ip}
	resp := make(chan struct{}, 1)
	aliasStr := &types.Alias{Device: dev, Name: alias}
	log.Debugf("Alias is %v", &aliasStr)
	aliasResp := &types.AliasResponse{Alias: *aliasStr, Response: resp}
	deviceChan <- aliasResp

	if _, ok := <-resp; !ok {
		return nil, errors.New("Error adding device")
	}
	return aliasStr, nil
}

func getAllAliases() []string {
	aliasChan := make(chan string)
	aliasRequestChan <- aliasChan
	aliases := make([]string, 0, 0)

	for alias := range aliasChan {
		aliases = append(aliases, alias)
	}
	return aliases
}

func getAllDevices() map[string]*types.Device {
	aliases := getAllAliases()
	devices := make(map[string]*types.Device)

	for _, alias := range aliases {
		dev, err := getDevice(alias)
		if err != nil {
			log.Errorf("Got error retrieving device %s, cause: %v", alias, err)
		}
		devices[alias] = dev
	}
	return devices
}

func getDevice(alias string) (*types.Device, error) {
	response := make(chan *types.Device)
	getDev := &types.GetDev{Alias: alias, Response: response}

	getChan <- getDev
	device := <-response
	close(response)

	if device == nil {
		return device, errors.New("No such device")
	}

	return device, nil
}
