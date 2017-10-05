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
)

const delims = ":-"

var initialized = false
var deviceChan = make(chan *types.Alias)
var getChan = make(chan *types.GetDev)
var passHandlingChan = make(chan *types.PasswordHandling)
var updatePassChan = make(chan *types.PasswordUpdate)
var reMAC = regexp.MustCompile(`^([0-9a-fA-F]{2}[` + delims + `]){5}([0-9a-fA-F]{2})$`)
var ifaceList = make([]string, 0, 0)

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
}

func handleDevices(w http.ResponseWriter, r *http.Request) {
	if !initialized {
		redirectToConfig(w, r)
		return
	}
	switch r.Method {
	case "GET":

	case "POST":
	default:
		handleError(w, r, errors.New("Not Allowed"), 405)
	}
}

func redirectToConfig(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/config", 301)
}

func config(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET": //Got first request, sending back page
		templ, err := template.ParseFiles("templates/config.html")
		templ = template.Must(templ, err)
		templ.Execute(w, nil)
	case "POST": //Got submit running it!!!
		password := r.FormValue("password")
		if password == "" {
			handleError(w, r, errors.New("Empty Password"), 422)
			return
		}
		go storage.StartHandling(password, deviceChan, getChan, passHandlingChan, updatePassChan)
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

func registerDevice(alias, mac, iface string) error {
	if !reMAC.MatchString(mac) {
		return errors.New("Invalid mac address format")
	}

	dev := &types.Device{Iface: iface, Mac: mac}
	resp := make(chan struct{}, 1)
	aliasStr := &types.Alias{Device: dev, Name: alias, Response: resp}
	deviceChan <- aliasStr

	if _, ok := <-resp; !ok {
		return errors.New("Error adding device")
	}
	return nil
}

func sendPacket(computerName, localIface string) error {
	macAddr, bcastAddr := "ciaone", "ciaone"
	err := wol.SendMagicPacket(macAddr, bcastAddr, localIface)
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
