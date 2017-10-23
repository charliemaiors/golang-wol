package types

import "time"

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
}

//WakeUpReport represent a report after wake up attempt
type WakeUpReport struct {
	Alias  string
	Report map[time.Time]bool
}
