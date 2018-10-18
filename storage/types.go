package storage

import (
	"time"

	"github.com/charliemaiors/golang-wol/types"
)

type ActionSubject struct {
	Action types.Action
	Alias  string
}

type DeviceAccessor interface {
	AddDevice(dev *types.Device, name string) error
	GetDevice(alias string) (*types.Device, error)
	GetAllAliases() []string
	DeleteDevice(alias string) error
}

type PasswordAccessor interface {
	CheckPassword(password string) error
	UpdatePassword(oldpass, newpass string) error
}

type SchedulerAccessor interface {
	ScheduleOperation(alias string, action types.Action, when time.Time) error
	GetAllOperations(aliases []string) map[ActionSubject]time.Time
}
