package server

import (
	"errors"
	"html/template"
	"net"
	"net/http"
	"regexp"

	"bitbucket.org/cmaiorano/golang-wol/storage"
	"bitbucket.org/cmaiorano/golang-wol/types"
	wol "github.com/sabhiram/go-wol"
	ping "github.com/tatsushid/go-fastping"
)

const delims = ":-"

var initialized = false
var deviceChan = make(chan *types.Alias)
var getChan = make(chan *types.GetDev)
var passHandlingChan = make(chan *types.PasswordHandling)
var updatePassChan = make(chan *types.PasswordUpdate)
var aliasRequestChan = make(chan chan string)
var reMAC = regexp.MustCompile(`^([0-9a-fA-F]{2}[` + delims + `]){5}([0-9a-fA-F]{2})$`)
var ifaceList = make([]string, 0, 0)
var pinger *ping.Pinger

func init() {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, v := range ifaces {
		ifaceList = append(ifaceList, v.Name)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if !initialized {
		redirectToConfig(w, r)
		return
	}

	switch r.Method {
	case "GET":
		templ, err := template.ParseFiles("../templates/index.gohtml")
		templ = template.Must(templ, err)
		aliases := getAllAliases()
		templ.Execute(w, aliases)
	case "POST":
		handleRootPost(w, r)
	default:
		handleError(w, r, errors.New("Method not allowed"), 405)
	}
}

func handleDevices(w http.ResponseWriter, r *http.Request) {
	if !initialized {
		redirectToConfig(w, r)
		return
	}
	switch r.Method {
	case "GET":
		templ, err := template.ParseFiles("templates/add-device.gohtml")
		templ = template.Must(templ, err)
		err = templ.Execute(w, ifaceList)
		if err != nil {
			panic(err)
		}
	case "POST":
		handleDevicePost(w, r)
		return
	default:
		handleError(w, r, errors.New("Not Allowed"), 405)
	}
}

func redirectToConfig(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/config", 301)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET": //Got first request, sending back page
		templ, err := template.ParseFiles("templates/config.html")
		templ = template.Must(templ, err)
		templ.Execute(w, nil)
	case "POST": //Got submit running it!!!
		err := r.ParseForm()
		if err != nil {
			handleError(w, r, err, 422)
			return
		}
		password := r.FormValue("password")
		if password == "" {
			handleError(w, r, errors.New("Empty Password"), 422)
			return
		}
		go storage.StartHandling(password, deviceChan, getChan, passHandlingChan, updatePassChan, aliasRequestChan)
		initialized = true
		templ, err := template.ParseFiles("templates/config-success.html")
		templ = template.Must(templ, err)
		err = templ.Execute(w, nil)
		if err != nil {
			panic(err)
		}
	default:
		handleError(w, r, errors.New("Not Allowed"), 405)
	}
}

func handleRootPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		handleError(w, r, err, 422)
		return
	}

	err = checkPassword(r.FormValue("password"))
	if err != nil {
		handleError(w, r, err, 401)
		return
	}
}

func handleDevicePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		handleError(w, r, err, 422)
		return
	}

	err = checkPassword(r.FormValue("password"))

	if err != nil {
		handleError(w, r, err, 401)
		return
	}

	alias, regErr := registerDevice(r.FormValue("alias"), r.FormValue("macAddr"), r.FormValue("ifaces"), r.FormValue("ipAddr"))
	if regErr != nil {
		handleError(w, r, err, 422)
		return
	}

	templ, err := template.ParseFiles("templates/add-device-success.gohtml")
	templ = template.Must(templ, err)
	templ.Execute(w, alias)
}

func checkPassword(password string) error {
	respChan := make(chan error)
	pass := &types.PasswordHandling{Password: password, Response: respChan}
	passHandlingChan <- pass
	err := <-respChan
	return err
}

func registerDevice(alias, mac, iface, ip string) (*types.Alias, error) {
	if !reMAC.MatchString(mac) {
		return nil, errors.New("Invalid mac address format")
	}

	dev := &types.Device{Iface: iface, Mac: mac, IP: ip}
	resp := make(chan struct{}, 1)
	aliasStr := &types.Alias{Device: dev, Name: alias, Response: resp}
	deviceChan <- aliasStr

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

func getDevice(alias string) (*types.Device, error) {
	response := make(chan *types.Device)
	getDev := &types.GetDev{Alias: alias, Response: response}

	getChan <- getDev
	device := <-response

	if device == nil {
		return device, errors.New("No such device")
	}

	return device, nil
}

func sendPacket(computerName string) error {
	dev, err := getDevice(computerName)

	if err != nil {
		return err
	}

	err = wol.SendMagicPacket(dev.Mac, "255.255.255.0", dev.Iface)
	return err
}

func handleError(w http.ResponseWriter, r *http.Request, err error, errCode int) {
	response := types.ResponseError{
		Message: err.Error(),
	}
	w.WriteHeader(errCode)
	t, err := template.ParseFiles("templates/error.gohtml")
	t = template.Must(t, err)
	t.Execute(w, response)
}
