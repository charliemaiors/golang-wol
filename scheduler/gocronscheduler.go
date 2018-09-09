package scheduler

import (
	"fmt"

	"github.com/jasonlvhit/gocron"
)

type GoCronScheduler struct {
}

func (scheduler *GoCronScheduler) Schedule(hour, minutes, device string) {
	gocron.Every(1).Day().Do(fmt.Printf)
}
