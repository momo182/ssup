package outrig

import "github.com/outrigdev/outrig"

func Push(name string, value any) {
	pusher := outrig.NewWatch(name).AsGoFmt().ForPush()
	pusher.Push(value)
	pusher.Unregister()
}

func String(name, msg string) {
	val := msg
	pusher := outrig.NewWatch(name).Static(val)
	pusher.Unregister()
}
