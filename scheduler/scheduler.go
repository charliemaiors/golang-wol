package scheduler

type Action int

const (
	WakeUp Action = iota
	Shutdown
)

type Scheduler interface {
	Schedule(hour, minutes, device string, action Action) error
}
