package server

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"regexp"
	"strings"

	rice "github.com/GeertJohan/go.rice"
	"github.com/charliemaiors/golang-wol/storage"
	"github.com/charliemaiors/golang-wol/types"
	"github.com/charliemaiors/golang-wol/utils"
	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

const (
	delims = ":-"
)

var (
	initialized      = false
	deviceChan       = make(chan *types.AliasResponse)
	getChan          = make(chan *types.GetDev)
	passHandlingChan = make(chan *types.PasswordHandling)
	updatePassChan   = make(chan *types.PasswordUpdate)
	delDevChan       = make(chan *types.DelDev)

	devStatus        = make(map[string]bool) //represent a sort of cache used for device status retrieve
	aliasRequestChan = make(chan chan string)
	reMAC            = regexp.MustCompile(`^([0-9a-fA-F]{2}[` + delims + `]){5}([0-9a-fA-F]{2})$`)
	templateBox      *rice.Box
	solcommand       string
	turnOffPort      string
	prefix           = ""
	handler          http.Handler
)

func init() {
	log.SetLevel(log.DebugLevel)

}

func configureBox() {
	var err error
	templateBox, err = rice.FindBox("../templates")
	if err != nil {
		panic(err)
	}
	configRouter()
}

func handleProxy() {
	handler = handlers.ProxyHeaders(handler)
}

func configRouter() {
	router := httprouter.New()
	router.HandleMethodNotAllowed = true
	router.MethodNotAllowed = handleNotAllowed
	router.NotFound = handleNotFound

	router.GET(prefix+"/manage-dev", handleManageDevicesGet)
	router.POST(prefix+"/manage-dev", handleManageDevicePost)

	router.GET(prefix+"/devices", handleDevicesGet)
	router.POST(prefix+"/devices/:alias", handleDevicePost)
	router.GET(prefix+"/devices/:alias", handleDeviceGet)
	router.DELETE(prefix+"/devices/:alias", handleDeviceDelete)

	router.GET(prefix+"/", handleRootGet)
	router.POST(prefix+"/", handleRootPost)

	router.POST(prefix+"/ping/:alias", handlePing)

	router.GET(prefix+"/config", handleConfigGet)
	router.POST(prefix+"/config", handleConfigPost)

	handler = router
}

func handleManageDevicesGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !initialized {
		http.Redirect(w, r, prefix+"/config", http.StatusTemporaryRedirect)
		return
	}
	tmpbl, err := templateBox.String("add-device.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("addDev").Parse(tmpbl))
	err = templ.Execute(w, prefix)
	if err != nil {
		panic(err)
	}
}

func handleManageDevicePost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !initialized {
		http.Redirect(w, r, prefix+"/config", http.StatusTemporaryRedirect)
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
	aliasDef := types.DevPageAlias{Alias: *alias, Prefix: prefix}
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
	templ.Execute(w, aliasDef)
}

func handleDevicesGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	devices := getAllDevices()
	tmpbl, err := templateBox.String("device-list.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templateStruct := DeviceListRevProxy{Devices: devices, Prefix: prefix}
	templ := template.Must(template.New("devsGet").Parse(tmpbl))
	templ.Execute(w, templateStruct)
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
	devPageAlias := types.DevPageAlias{Prefix: prefix, Alias: alias}

	tmpbl, err := templateBox.String("device.gohtml")
	if err != nil {
		log.Errorf("Got error getting template %v", err)
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}
	templ := template.Must(template.New("devGet").Parse(tmpbl))
	templ.Execute(w, devPageAlias)
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
	aliasFull := types.DevPageAlias{Alias: *aliasType, Prefix: prefix}

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
	templ.Execute(w, aliasFull)
}

func handlePing(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	alias := ps.ByName("alias")
	log.Debugf("Current alias is %s", alias)

	dev, err := getDevice(alias)

	if err != nil {
		handleError(w, r, err, http.StatusNotFound)
		return
	}

	alive := utils.CheckHealt(dev.IP)

	resp := struct {
		Message bool `json:"message"`
	}{}

	resp.Message = alive
	devStatus[alias] = alive

	enc := json.NewEncoder(w)
	err = enc.Encode(&resp)

	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
		return
	}
}

func handleConfigGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if initialized {
		tmpbl, err := templateBox.String("config-update.gohtml")
		if err != nil {
			handleError(w, r, err, http.StatusUnprocessableEntity)
		}
		templ := template.Must(template.New("conf-upd").Parse(tmpbl))
		templ.Execute(w, prefix)
	} else {
		tmpbl, err := templateBox.String("config.gohtml")
		if err != nil {
			handleError(w, r, err, http.StatusUnprocessableEntity)
		}
		templ := template.Must(template.New("conf").Parse(tmpbl))
		templ.Execute(w, prefix)
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
	configSuccess := ConfigSuccess{AlreadyInit: false, Prefix: prefix}
	templ := template.Must(template.New("confSucc").Parse(tmpbl))
	err = templ.Execute(w, configSuccess)
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

	configSuccess := ConfigSuccess{AlreadyInit: true, Prefix: prefix}
	templ := template.Must(template.New("confUpd").Parse(tmpbl))
	err = templ.Execute(w, configSuccess)
	if err != nil {
		panic(err)
	}
}

func handleRootGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !initialized {
		http.Redirect(w, r, prefix+"/config", http.StatusTemporaryRedirect)
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

	alive := devStatus[r.FormValue("devices")]

	err = handleDeviceAction(alive, dev)
	if err != nil {
		log.Errorf("Got error handling device %v", err)
		handleError(w, r, err, http.StatusBadRequest)
		return
	}

	log.Debugf("Packet sent, now waiting for wake up")
	report, pingErr := utils.PingHost(dev.IP, alive)
	if pingErr != nil {
		log.Errorf("Got error %v pinging, the executables has right capacity? if no use setcap cap_net_raw=+ep golang-wol", pingErr)
		handleError(w, r, pingErr, http.StatusInternalServerError)
		return
	}
	wakeupRep := &types.Report{Alias: r.FormValue("devices"), Alive: alive, Report: report}
	tmpbl, err := templateBox.String("report.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("rep").Parse(tmpbl))
	templ.Execute(w, wakeupRep)
}

func handleDeviceAction(alive bool, dev *types.Device) error {
	if alive {
		log.Debugf("Device %v, sending command", dev)
		err := utils.TurnOffDev(dev.IP, turnOffPort, solcommand)
		if err != nil {
			log.Errorf("Got error sending command %v", err)
			return err
		}
	} else {
		log.Debugf("Device %v, sending packets", dev)
		err := utils.SendPacket(dev.Mac, dev.IP)
		if err != nil {
			log.Errorf("Got error sending packets %v", err)
			return err
		}
	}
	return nil
}

func handleError(w http.ResponseWriter, r *http.Request, err error, errCode int) {
	response := types.ResponseError{
		Message: err.Error(),
		Prefix:  prefix,
	}
	w.WriteHeader(errCode)
	tmpbl, err := templateBox.String("error.gohtml")
	if err != nil {
		handleError(w, r, err, http.StatusUnprocessableEntity)
	}
	templ := template.Must(template.New("error").Parse(tmpbl))
	templ.Execute(w, response)
}
