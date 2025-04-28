//go:build darwin
// +build darwin

package qemuguestagent

import (
	sleepnotifier "github.com/prashantgupta24/mac-sleep-notifier/notifier"
)

func StartSleepNotifier() <-chan SleepNotifierActivityType {
	sleepNotifierCh := sleepnotifier.GetInstance().Start()
	sleepNotifierActivityCh := make(chan SleepNotifierActivityType)
	go func() {
		defer close(sleepNotifierActivityCh)
		for activity := range sleepNotifierCh {
			switch activity.Type {
			case sleepnotifier.Awake:
				sleepNotifierActivityCh <- SleepNotifierActivityTypeAwake
			case sleepnotifier.Sleep:
				sleepNotifierActivityCh <- SleepNotifierActivityTypeSleep
			}
		}
	}()
	return sleepNotifierActivityCh
}
