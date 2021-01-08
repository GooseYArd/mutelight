package main

// Trivial source mute indicator using a Thingm Blink(1)
//
// TODO:
//  pidfile
//  accept source device index as a flag

import (
	"errors"
	"github.com/godbus/dbus"
	"github.com/todbot/go-blink1"
	"github.com/sqp/pulseaudio"
	"log"
        "os"
        "syscall"
)

var quit = make(chan struct{})
var state_chan = make(chan muteState)
var sock_path = "/tmp/mutelight.sock"

// The command I bind to a key in xmonad to mute my mic is
// pactl set-source-mute 1 toggle

// Adjust this to the source you want to mute
var target_device = dbus.ObjectPath("/org/pulseaudio/core1/source1")

func PreparePipe(path string) {
	pipeExists := false
	fileInfo, err := os.Stat(path)

	if err == nil {
		if (fileInfo.Mode() & os.ModeNamedPipe) > 0 {
			pipeExists = true
		} else {
			log.Printf("%d != %d\n", os.ModeNamedPipe, fileInfo.Mode())
			panic(path + " exists, but it's not a named pipe (FIFO)")
		}
	}

	// Try to create pipe if needed
	if !pipeExists {
		err := syscall.Mkfifo(path, 0666)
		if err != nil {
			panic(err.Error())
		}
	}
}

func WriteState(pipe string, muted bool) {
    sock, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0600)
    testFatal(err, "Unable to open status pipe")
    defer sock.Close()
    if muted {
        sock.Write([]byte("\n"));
    } else {
        sock.Write([]byte("!!! NOT MUTED !!!\n"))
    }
}

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

        PreparePipe(sock_path)

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
        WriteState(sock_path, muted)

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
                                        WriteState(sock_path, state.muted)
				} else {
					color := blink1.State{
						Red: 255,
					}
					device.SetState(color)
                                        WriteState(sock_path, state.muted)
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
