package storage

import (
	"time"

	"github.com/charliemaiors/golang-wol/types"
)

type DeviceAccessor interface {
	AddDevice(dev *types.Device, name string)
	GetDevice(alias string) *types.Device
	DeleteDevice(alias string) error
}

type PasswordAccessor interface {
	CheckPassword(password string) error
	UpdatePassword(oldpass, newpass string) error
}

type SchedulerAccessor interface {
	ScheduleOperation(alias string, action types.Action, when time.Time) error
}
