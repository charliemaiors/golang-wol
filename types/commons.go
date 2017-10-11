package types

//Device is the simple rapresentation of a remote device target of wake up
type Device struct {
	Mac   string `json:"mac"`
	Iface string `json:"iface"`
}

//Alias is the full structure of device plus a common name used as alias
type Alias struct {
	Device   *Device
	Name     string
	Response chan struct{}
}

//GetDev is an internal object used for api
type GetDev struct {
	Alias    string
	Response chan *Device
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

type ResponseError struct {
	Message string
}
