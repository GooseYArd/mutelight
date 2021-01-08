package main

import (
	"errors"
	"github.com/godbus/dbus"
	"github.com/hink/go-blink1"
	"github.com/sqp/pulseaudio"
	"log"
)

var quit = make(chan struct{})
var state_chan = make(chan muteState)

// The command I bind to a key in xmonad to mute my mic is
// pactl set-source-mute 1 toggle

// Adjust this to the source you want to mute
var target_device = dbus.ObjectPath("/org/pulseaudio/core1/source1")

type muteState struct {
	muted bool
}

func main() {
	isLoaded, e := pulseaudio.ModuleIsLoaded()
	testFatal(e, "test pulse dbus module is loaded")
	if !isLoaded {
		e = pulseaudio.LoadModule()
		testFatal(e, "load pulse dbus module")
		defer pulseaudio.UnloadModule()
	}

	pulse, e := pulseaudio.New()
	testFatal(e, "connect to the pulse service")
	defer pulse.Close()

	app := &AppPulse{}
	pulse.Register(app)
	defer pulse.Unregister(app)

	muted, err := is_muted(pulse, target_device)
	testFatal(err, "couldn't find target device")

	device, err := blink1.OpenNextDevice()
	defer device.Close()
	if muted {
		color := blink1.State{}
		device.SetState(color)
	} else {
		color := blink1.State{
			Red: 255,
		}
		device.SetState(color)
	}

	go func() {
		if err != nil {
			panic(err)
		}
		for {
			select {
			case state := <-state_chan:
				if state.muted {
					color := blink1.State{}
					device.SetState(color)
				} else {
					color := blink1.State{
						Red: 255,
					}
					device.SetState(color)
				}
			}
		}
	}()

	go pulse.Listen()
	defer pulse.StopListening()

	<-quit
}

func is_muted(client *pulseaudio.Client, path dbus.ObjectPath) (bool, error) {
	sources, e := client.Core().ListPath("Sources")
	testFatal(e, "get list of sinks")

	if len(sources) == 0 {
		return false, errors.New("no sources!")
	}

	dev := client.Device(sources[1])
	mute, _ := dev.Bool("Mute")
	return mute, nil
}

type AppPulse struct{}

func (ap *AppPulse) DeviceMuteUpdated(path dbus.ObjectPath, state bool) {
	if path == target_device {
		state_msg := muteState{
			muted: state,
		}
		state_chan <- state_msg
	}
}

func testFatal(e error, msg string) {
	if e != nil {
		log.Fatalln(msg+":", e)
	}
}
