package server

import (
	b64 "encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"html/template"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/charliemaiors/golang-wol/storage"
	"github.com/charliemaiors/golang-wol/types"
	"github.com/julienschmidt/httprouter"
	wol "github.com/sabhiram/go-wol"
	log "github.com/sirupsen/logrus"
	ping "github.com/tatsushid/go-fastping"
)

const delims = ":-"

var (
	initialized      = false
	deviceChan       = make(chan *types.AliasResponse)
	getChan          = make(chan *types.GetDev)
	passHandlingChan = make(chan *types.PasswordHandling)
	updatePassChan   = make(chan *types.PasswordUpdate)
	delDevChan       = make(chan *types.DelDev)

	aliasRequestChan = make(chan chan string)
	reMAC            = regexp.MustCompile(`^([0-9a-fA-F]{2}[` + delims + `]){5}([0-9a-fA-F]{2})$`)
	ifaceList        = make([]string, 0, 0)
	pinger           *ping.Pinger
	templateBox      *rice.Box
	router           *httprouter.Router
)

func init() {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, v := range ifaces {
		ifaceList = append(ifaceList, v.Name)
	}

	pinger = ping.NewPinger()
	log.SetLevel(log.DebugLevel)
	configRouter()
}

func loadBox() {
	var err error
	templateBox, err = rice.FindBox("../templates")
	if err != nil {
		panic(err)
	}
	log.Debugf("Is embedded? %v", templateBox.IsEmbedded())
}

func configRouter() {
	router = httprouter.New()
	router.HandleMethodNotAllowed = true
	router.MethodNotAllowed = handleNotAllowed
	router.NotFound = handleNotFound

	router.GET("/manage-dev", handleManageDevicesGet)
	router.POST("/manage-dev", handleManageDevicePost)

	router.GET("/devices", handleDevicesGet)
	router.POST("/devices/:alias", handleDevicePost)
	router.GET("/devices/:alias", handleDeviceGet)
	router.DELETE("/devices/:alias", handleDeviceDelete)

	router.GET("/", handleRootGet)
	router.POST("/", handleRootPost)

	router.GET("/config", handleConfigGet)
	router.POST("/config", handleConfigPost)
}

func handleManageDevicesGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !initialized {
		http.Redirect(w, r, "/config", http.StatusTemporaryRedirect)
		return
	}
	tmpbl, err := templateBox.String("add-device.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("addDev").Parse(tmpbl))
	err = templ.Execute(w, ifaceList)
	if err != nil {
		panic(err)
	}
}

func handleManageDevicePost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !initialized {
		http.Redirect(w, r, "/config", http.StatusTemporaryRedirect)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Errorf("Error parsing form %v", err)
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}

	err = checkPassword(r.FormValue("password"))

	if err != nil {
		handleError(w, r, err, http.StatusUnauthorized)
		return
	}

	alias, regErr := registerOrUpdateDevice(r.FormValue("alias"), r.FormValue("macAddr"), r.FormValue("ipAddr"))
	if regErr != nil {
		log.Errorf("Error registering %v", regErr)
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}

	tmpbl, err := templateBox.String("add-device-success.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}
	templ := template.Must(template.New("addDevSucc").Parse(tmpbl))
	templ.Execute(w, alias)
}

func handleDevicesGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	devices := getAllDevices()
	tmpbl, err := templateBox.String("device-list.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("devsGet").Parse(tmpbl))
	templ.Execute(w, devices)
}

func handleDeviceGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	deviceName := ps.ByName("alias")
	log.Debugf("Getting data for %s", deviceName)
	dev, err := getDevice(deviceName)
	if err != nil {
		log.Errorf("Got error finding device %v", err)
		handleError(w, r, err, http.StatusNotFound)
		return
	}
	alias := types.Alias{Device: dev, Name: deviceName}

	tmpbl, err := templateBox.String("device.gohtml")
	if err != nil {
		log.Errorf("Got error getting template %v", err)
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}
	templ := template.Must(template.New("devGet").Parse(tmpbl))
	templ.Execute(w, alias)
}

func handleDeviceDelete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	alias := ps.ByName("alias")
	log.Debugf("Got device delete request for %s", alias)
	token := r.Header.Get("X-Auth-Token")
	pass, err := b64.StdEncoding.DecodeString(token)
	if err != nil {
		log.Errorf("Got error %v", err)
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}

	err = checkPassword(string(pass))
	if err != nil {
		log.Errorf("Got error %v", err)
		handleError(w, r, err, http.StatusUnauthorized)
		return
	}

	err = delDevice(alias)
	if err != nil {
		log.Errorf("Got error %v", err)
		handleError(w, r, err, http.StatusBadGateway)
		return
	}

	log.Debug("Everything went fine, responding")
	respJs := struct {
		Message string `json:"message"`
	}{Message: alias + " removed"}
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(respJs)
}

func handleDevicePost(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	alias := ps.ByName("alias")
	err := r.ParseForm()
	if err != nil {
		log.Errorf("Something bad happened with form %v", err)
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}
	formAlias := r.FormValue("alias")
	err = checkPassword(r.FormValue("password"))

	if err != nil {
		log.Errorf("Wrong password? %v", err)
		handleError(w, r, err, http.StatusUnauthorized)
		return
	}

	if strings.Compare(alias, formAlias) != 0 {
		err = delDevice(alias) //Delete post target device in order to create a new one in db
		if err != nil {
			log.Errorf("Got error deleting device %v", err)
			handleError(w, r, err, http.StatusBadRequest)
			return
		}
	}
	aliasType, err := registerOrUpdateDevice(formAlias, r.FormValue("macAddr"), r.FormValue("ipAddr")) //updating device or creating a new one

	if err != nil {
		log.Errorf("Wrong password? %v", err)
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}

	tmpbl, err := templateBox.String("updated-device-success.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}
	templ := template.Must(template.New("addDevSucc").Parse(tmpbl))
	templ.Execute(w, aliasType)
}

func handleConfigGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if initialized {
		tmpbl, err := templateBox.String("config-update.html")
		if err != nil {
			handleError(w, r, err, http.StatusUnprocessableEntity)
		}
		templ := template.Must(template.New("conf-upd").Parse(tmpbl))
		templ.Execute(w, nil)
	} else {
		tmpbl, err := templateBox.String("config.html")
		if err != nil {
			handleError(w, r, err, http.StatusUnprocessableEntity)
		}
		templ := template.Must(template.New("conf").Parse(tmpbl))
		templ.Execute(w, nil)
	}
}

func handleConfigPost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := r.ParseForm()
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}

	if initialized {
		handleConfigUpdate(w, r)
	} else {
		handleConfigInit(w, r)
	}
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, errors.New("Not Found"), http.StatusNotFound)
}

func handleNotAllowed(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, errors.New("Method not Allowed"), http.StatusMethodNotAllowed)
}

func handleConfigInit(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	if password == "" {
		handleError(w, r, errors.New("Empty Password"), http.StatusUnprocessableEntity)
		return
	}
	storage.InitLocal(password)
	go storage.StartHandling(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, aliasRequestChan)

	initialized = true
	tmpbl, err := templateBox.String("config-success.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("confSucc").Parse(tmpbl))
	err = templ.Execute(w, false)
	if err != nil {
		panic(err)
	}
}

func handleConfigUpdate(w http.ResponseWriter, r *http.Request) {

	oldPass := r.FormValue("oldPassword")
	newPass := r.FormValue("newPassword")

	if newPass == "" || oldPass == "" {
		log.Errorf("New pass %s OldPass %s", newPass, oldPass)
		handleError(w, r, errors.New("One of the correspondent passwords are empty"), http.StatusUnprocessableEntity)
		return
	}
	resp := make(chan error)
	passUpdate := &types.PasswordUpdate{NewPassword: newPass, OldPassword: oldPass, Response: resp}
	updatePassChan <- passUpdate

	err, ok := <-resp
	if ok && err != nil {
		log.Errorf("Got error %v", err)
		handleError(w, r, err, http.StatusBadGateway)
		return
	}

	tmpbl, err := templateBox.String("config-success.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("confUpd").Parse(tmpbl))
	err = templ.Execute(w, true)
	if err != nil {
		panic(err)
	}
}

func handleRootGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !initialized {
		http.Redirect(w, r, "/config", http.StatusTemporaryRedirect)
		return
	}
	tmpbl, err := templateBox.String("index.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("index").Parse(tmpbl))
	aliases := getAllAliases()
	templ.Execute(w, aliases)
}

func handleRootPost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("Got error parsing form %v", err)
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}

	log.Debug("Form parsed")
	err = checkPassword(r.FormValue("password"))
	if err != nil {
		log.Errorf("Got error checking password %v", err)
		handleError(w, r, err, http.StatusUnauthorized)
		return
	}

	log.Debug("Password valid, getting target device")
	dev, err := getDevice(r.FormValue("devices"))
	if err != nil {
		log.Errorf("No device error: %v", err)
		handleError(w, r, err, http.StatusNotFound)
		return
	}

	log.Debugf("Found device %v, sending packets", dev)
	err = sendPacket(dev.Mac)
	if err != nil {
		log.Errorf("Got error sending packets %v", err)
		handleError(w, r, err, http.StatusInternalServerError)
		return
	}

	log.Debugf("Packet sent, now waiting for wake up")
	report, pingErr := pingHost(dev.IP)
	if pingErr != nil {
		log.Errorf("Got error %v pinging, the executables has right capacity? if no use setcap cap_net_raw=+ep golang-wol", pingErr)
		handleError(w, r, pingErr, http.StatusInternalServerError)
		return
	}
	wakeupRep := &types.WakeUpReport{Alias: r.FormValue("devices"), Report: report}
	tmpbl, err := templateBox.String("report.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("rep").Parse(tmpbl))
	templ.Execute(w, wakeupRep)
}

func pingHost(ip string) (map[time.Time]bool, error) {
	pinger.AddIP(ip)
	defer pinger.RemoveIP(ip)
	stopped := false
	report := make(map[time.Time]bool)
	pinger.OnIdle = func() {
		report[time.Now()] = false
	}

	pinger.OnRecv = func(ip *net.IPAddr, tdur time.Duration) {
		report[time.Now()] = true
		log.Debugf("Got answer from %v", ip.String())
		stopped = true
		pinger.Stop()
	}

	pinger.RunLoop()
	ticker := time.NewTicker(time.Millisecond * 30)
	select {
	case <-pinger.Done():
		if err := pinger.Err(); err != nil {
			log.Errorf("Ping failed: %v", err)
			return nil, err
		}
		log.Debugf("Got stop for ping alive!!!")
	case <-ticker.C:
		break
	}
	ticker.Stop()
	if !stopped {
		pinger.Stop()
	}
	return report, nil
}

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

	if device == nil {
		return device, errors.New("No such device")
	}

	return device, nil
}

func sendPacket(mac string) error {
	err := wol.SendMagicPacket(mac, "", "")
	return err
}

func handleError(w http.ResponseWriter, r *http.Request, err error, errCode int) {
	response := types.ResponseError{
		Message: err.Error(),
	}
	w.WriteHeader(errCode)
	tmpbl, err := templateBox.String("error.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("error").Parse(tmpbl))
	templ.Execute(w, response)
}

func getBcastAddr(ipAddr string) (string, error) { // works when the n is a prefix, otherwise...

	ipParsed := net.ParseIP(ipAddr)
	mask := ipParsed.DefaultMask()
	log.Debugf("Passed ip: %s, ipParsed: %v, mask: %v", ipAddr, ipParsed, mask)

	n := &net.IPNet{IP: ipParsed, Mask: mask}
	log.Debugf("IpNet: %v", n)
	if n.IP.To4() == nil {
		return "", errors.New("does not support IPv6 addresses")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip.String(), nil
}

func checkIfFolderExist(loc string) error {
	info, err := os.Stat(loc)
	if os.IsNotExist(err) {
		err = os.MkdirAll(loc, os.ModeDir)
		return err
	} else if !info.IsDir() {
		return errors.New("Exist but is not a folder")
	}
	return nil
}
