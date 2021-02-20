package main

// Trivial source mute indicator using a Thingm Blink(1)
//
// TODO:
//  pidfile
//  accept source device index as a flag

import (
	"errors"
	"github.com/godbus/dbus"
	"github.com/sqp/pulseaudio"
	"github.com/todbot/go-blink1"
	"log"
	"os"
	"syscall"
)

var quit = make(chan struct{})
var stateChan = make(chan muteState)
var sockPath = "/tmp/mutelight.sock"

// The command I bind to a key in xmonad to mute my mic is
// pactl set-source-mute 1 toggle

// Adjust this to the source you want to mute
var deviceName = "alsa_input.usb-BEHRINGER_UMC202HD_192k-00.analog-stereo"

func preparePipe(path string) {
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

func writeState(pipe string, muted bool) {
	sock, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0600)
	testFatal(err, "Unable to open status pipe")
	defer sock.Close()
	if muted {
		sock.Write([]byte("\n"))
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

        preparePipe(sockPath)

        app := &appPulse{Client: pulse, targetDevice: deviceName}
	pulse.Register(app)
	defer pulse.Unregister(app)

	muted, err := isMuted(pulse, deviceName)
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
	writeState(sockPath, muted)

	go func() {
		if err != nil {
			panic(err)
		}
		for {
			select {
			case state := <-stateChan:
				if state.muted {
					color := blink1.State{}
					device.SetState(color)
					writeState(sockPath, state.muted)
				} else {
					color := blink1.State{
						Red: 255,
					}
					device.SetState(color)
					writeState(sockPath, state.muted)
				}
			}
		}
	}()

	go pulse.Listen()
	defer pulse.StopListening()

	<-quit
}


func isMuted(client *pulseaudio.Client, deviceName string) (bool, error) {
	sources, e := client.Core().ListPath("Sources")
	testFatal(e, "get list of sinks")

	if len(sources) == 0 {
            return false, errors.New("no sources")
	}

        for _, path := range sources {
            dev := client.Device(path)
            var name string
            dev.Get("Name", &name)
            if name == deviceName {
                mute, _ := dev.Bool("Mute")
                if mute {
                    log.Printf("Device %s muted", name)
                } else {
                    log.Printf("Device %s unmuted", name)
                }
                return mute, nil
            }
        }

        return false, nil
}

type appPulse struct{
    Client *pulseaudio.Client
    targetDevice string
}

func (ap *appPulse) DeviceMuteUpdated(path dbus.ObjectPath, state bool) {
        dev := ap.Client.Device(path)
        var name string
        dev.Get("Name", &name)
        if name == deviceName {
                if state {
                    log.Printf("Device %s muted", name)
                } else {
                    log.Printf("Device %s unmuted", name)
                }
            stageMsg := muteState{
                    muted: state,
            }
            stateChan <- stageMsg
        } else {
            log.Printf("%s != %s\n", path, ap.targetDevice)
        }

}

func testFatal(e error, msg string) {
	if e != nil {
		log.Fatalln(msg+":", e)
	}
}
