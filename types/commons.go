package types

type Device struct {
	Mac   string `json:"mac"`
	Iface string `json:"iface"`
}

type Alias struct {
	Device   *Device
	Name     string
	Response chan struct{}
}

type GetDev struct {
	Alias    string
	Response chan *Device
}

type PasswordHandling struct {
	Password string
	Response chan error
}

type PasswordUpdate struct {
	OldPassword string
	NewPassword string
	Response    chan error
}

type ResponseError struct {
	Message string
}
