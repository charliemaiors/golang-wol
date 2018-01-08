package utils

import (
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"os"
	"time"

	ping "github.com/charliemaiors/go-fastping"
	wol "github.com/sabhiram/go-wol"
	log "github.com/sirupsen/logrus"
)

var pinger *ping.Pinger

func init() {
	pinger = ping.NewPinger()
}

func SendPacket(mac, ip string) error {
	bCastIP, err := getBcastAddr(ip)
	if err != nil {
		return err
	}
	bCastAddr := bCastIP + ":9" //9 is the default port for wake on lan
	err = wol.SendMagicPacket(mac, bCastAddr, "")
	if err != nil {
		return err
	}
	return nil
}

func TurnOffDev(ip, turnOffPort, solcommand string) error {
	resp, err := http.Post("http://"+ip+":"+turnOffPort+"/"+solcommand, "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}
	return nil
}

func PingHost(ip string, alive bool) (map[time.Time]bool, error) {
	pinger.AddIP(ip)
	defer pinger.RemoveIP(ip)
	stopped := false
	report := make(map[time.Time]bool)
	if alive {
		pinger.OnIdle = func() {
			report[time.Now()] = true
			log.Debugf("No answer, the device is aslepp")
			stopped = true
			pinger.Stop()
		}

		pinger.OnRecv = func(ip *net.IPAddr, tdur time.Duration) {
			report[time.Now()] = false
		}
	} else {
		pinger.OnIdle = func() {
			report[time.Now()] = false
		}

		pinger.OnRecv = func(ip *net.IPAddr, tdur time.Duration) {
			report[time.Now()] = true
			log.Debugf("Got answer from %v", ip.String())
			stopped = true
			pinger.Stop()
		}
	}

	pinger.RunLoop()
	ticker := time.NewTicker(time.Second * 75)
	select {
	case <-pinger.Done():
		if err := pinger.Err(); err != nil {
			log.Errorf("Ping failed: %v", err)
			return nil, err
		}
		log.Debugf("Got stop for ping!!!")
	case <-ticker.C:
		break
	}
	ticker.Stop()
	if !stopped {
		pinger.Stop()
	}
	return report, nil
} //TODO refactor

func CheckHealt(ip string) bool {
	pinger.AddIP(ip)
	defer pinger.RemoveIP(ip)
	alive := false
	pinger.OnRecv = func(ip *net.IPAddr, tdur time.Duration) {
		log.Debugf("Device with ip %s is alive", ip)
		alive = true
		pinger.Stop()
	}
	pinger.OnIdle = func() {
		log.Debug("Terminated ping")
	}

	pinger.RunLoop()
	ticker := time.NewTicker(time.Second * 10)
	select {
	case <-pinger.Done():
		if err := pinger.Err(); err != nil {
			log.Errorf("Ping failed: %v", err)
			return false
		}
		log.Debugf("Got stop for ping alive!!!")
	case <-ticker.C:
		break
	}
	ticker.Stop()
	// if !alive {
	pinger.Stop()
	// }
	return alive
}

func CheckIfFolderExist(loc string) error {
	info, err := os.Stat(loc)
	if os.IsNotExist(err) {
		err = os.MkdirAll(loc, os.ModeDir)
		return err
	} else if !info.IsDir() {
		return errors.New("Exist but is not a folder")
	}
	return nil
}

func getBcastAddr(ipAddr string) (string, error) { // works when the n is a prefix, otherwise...

	ipParsed := net.ParseIP(ipAddr)
	mask := ipParsed.DefaultMask() //weak assumption, but server MUST be able to reach target address otherwise ping will fail
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
