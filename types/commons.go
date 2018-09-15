package types

import "time"

type Action int

const (
	WakeUp Action = iota
	TurnOff
)

//Device is the simple rapresentation of a remote device target of wake up
type Device struct {
	Mac string
	IP  string
}

//Alias is the full structure of device plus a common name used as alias
type Alias struct {
	Device *Device
	Name   string
}

//DevPageAlias is the full structure of device plus a common name used as alias and prefix
type DevPageAlias struct {
	Alias
	Prefix string
}

type AliasResponse struct {
	Alias
	Response chan struct{}
}

//GetDev is an internal object used for api
type GetDev struct {
	Alias    string
	Response chan *Device
}

//DelDev internal type used for device delete
type DelDev struct {
	Alias    string
	Response chan error
}

//PasswordHandling is a structure used for password matching
type PasswordHandling struct {
	Password string
	Response chan error
}

//PasswordUpdate is used for update current password
type PasswordUpdate struct {
	OldPassword string
	NewPassword string
	Response    chan error
}

//ResponseError retrieves particular response error
type ResponseError struct {
	Message string
	Prefix  string
}

//WakeUpReport represent a report after wake up attempt
type Report struct {
	Alias  string
	Alive  bool
	Report map[time.Time]bool
}

func (dev *Device) String() string {
	return "Mac " + dev.Mac + " IP " + dev.IP
}

func (alias *AliasResponse) String() string {
	return "Added alias with name " + alias.Alias.Name + " and device data " + alias.Device.String()
}
