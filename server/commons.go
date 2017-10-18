package server

//go:generate rice embed-go

import (
	"encoding/binary"
	"errors"
	"html/template"
	"net"
	"net/http"
	"regexp"
	"time"

	"bitbucket.org/cmaiorano/golang-wol/storage"
	"bitbucket.org/cmaiorano/golang-wol/types"
	rice "github.com/GeertJohan/go.rice"
	wol "github.com/sabhiram/go-wol"
	log "github.com/sirupsen/logrus"
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
var templateBox *rice.Box

func init() {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, v := range ifaces {
		ifaceList = append(ifaceList, v.Name)
	}

	templateBox, err = rice.FindBox("../templates/")
	if err != nil {
		panic(err)
	}
	pinger = ping.NewPinger()
	log.SetLevel(log.DebugLevel)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if !initialized {
		http.Redirect(w, r, "/config", 302)
		return
	}

	switch r.Method {
	case "GET":
		tmpbl, err := templateBox.String("index.gohtml")
		if err != nil {
			handleError(w, r, err, 422)
		}
		templ := template.Must(template.New("index").Parse(tmpbl))
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
		http.Redirect(w, r, "/config", 302)
		return
	}
	switch r.Method {
	case "GET":
		tmpbl, err := templateBox.String("add-device.gohtml")
		if err != nil {
			handleError(w, r, err, 422)
		}
		templ := template.Must(template.New("addDev").Parse(tmpbl))
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

func handleConfig(w http.ResponseWriter, r *http.Request) {
	log.Debugf("handleConfig - Initialized? %v", initialized)
	switch r.Method {
	case "GET": //Got first request, sending back page
		tmpbl, err := templateBox.String("config.html")
		if err != nil {
			handleError(w, r, err, 422)
		}
		templ := template.Must(template.New("conf").Parse(tmpbl))
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
		storage.InitLocal(password)
		go storage.StartHandling(deviceChan, getChan, passHandlingChan, updatePassChan, aliasRequestChan)

		initialized = true
		tmpbl, err := templateBox.String("config-success.html")
		if err != nil {
			handleError(w, r, err, 422)
		}
		templ := template.Must(template.New("confSucc").Parse(tmpbl))
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
		log.Errorf("Got error parsing form %v", err)
		handleError(w, r, err, 422)
		return
	}

	log.Debug("Form parsed")
	err = checkPassword(r.FormValue("password"))
	if err != nil {
		log.Errorf("Got error checking password %v", err)
		handleError(w, r, err, 401)
		return
	}

	log.Debug("Password valid, getting target device")
	dev, err := getDevice(r.FormValue("devices"))
	if err != nil {
		log.Errorf("No device error: %v", err)
		handleError(w, r, err, 404)
		return
	}

	log.Debugf("Found device %v, sending packets", dev)
	err = sendPacket(dev.Mac)
	if err != nil {
		log.Errorf("Got error sending packets %v", err)
		handleError(w, r, err, 500)
		return
	}

	log.Debugf("Packet sent, now waiting for wake up")
	report, pingErr := pingHost(dev.IP)
	if pingErr != nil {
		log.Errorf("Got error %v pinging, the executables has right capacity? if no use setcap cap_net_raw=+ep golang-wol", pingErr)
		handleError(w, r, pingErr, 500)
		return
	}
	wakeupRep := &types.WakeUpReport{Alias: r.FormValue("devices"), Report: report}
	tmpbl, err := templateBox.String("report.gohtml")
	if err != nil {
		handleError(w, r, err, 422)
	}
	templ := template.Must(template.New("rep").Parse(tmpbl))
	templ.Execute(w, wakeupRep)
}

func handleDevicePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("Error parsing form %v", err)
		handleError(w, r, err, 422)
		return
	}

	err = checkPassword(r.FormValue("password"))

	if err != nil {
		handleError(w, r, err, 401)
		return
	}

	alias, regErr := registerDevice(r.FormValue("alias"), r.FormValue("macAddr"), r.FormValue("ipAddr"))
	if regErr != nil {
		log.Errorf("Error registering %v", regErr)
		handleError(w, r, err, 422)
		return
	}

	tmpbl, err := templateBox.String("add-device-success.gohtml")
	if err != nil {
		handleError(w, r, err, 422)
	}
	templ := template.Must(template.New("addDevSucc").Parse(tmpbl))
	templ.Execute(w, alias)
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

func checkPassword(password string) error {
	respChan := make(chan error)
	pass := &types.PasswordHandling{Password: password, Response: respChan}
	passHandlingChan <- pass
	err := <-respChan
	return err
}

func registerDevice(alias, mac, ip string) (*types.Alias, error) {
	if !reMAC.MatchString(mac) {
		return nil, errors.New("Invalid mac address format")
	}

	dev := &types.Device{Mac: mac, IP: ip}
	resp := make(chan struct{}, 1)
	aliasStr := &types.Alias{Device: dev, Name: alias, Response: resp}
	log.Debugf("Alias is %v", &aliasStr)
	deviceChan <- aliasStr

	if _, ok := <-resp; !ok {
		return nil, errors.New("Error adding device")
	}
	return aliasStr, nil
}

func getAllAliases() []string {
	log.Debug("GETTING ALL ALIASES")
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
		handleError(w, r, err, 422)
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
