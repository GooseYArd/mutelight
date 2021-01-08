# Trivial Source Mute Monitor for Pulseaudio and the Thingm Blink(1)

After becoming annoyed by trying to use in-app mute controls for
videoconferences, I considered writing a simple xmobar or trayer application
that would show me the mute state my headset source, but after fooling around
with the idea for a while, I realized I wanted the indicator to be more
conspicuous, without taking up a lot of screen real estate.

A few people at my office have USB busy lights, so I looked at the options and
found the ThingM Blink(1) was inexpensive and seemed to have a library in all
the languages I care about, so I got the mark iii model and wired it up.

https://blink1.thingm.com/


After some experimentation with the C and Python bindings, I decided to write
a go version, since I've done too-little go and it'd be simple for someone
else who wanted to build this code if they wanted to use my hack as a starting
point for their own. Indeed, this code is mostly cut and pasted together
from the blink and pulseaudio examples.

This code is so trivial that I don't think it warrants much explanation, and I
barely know Go anyhow, but I will mention a few things I learned while hacking
on it. 

Also, I've made no attempt to make this configurable, for the same reason--
see the note below about the audio source index.

On Debian and Ubuntu you'll need libusb-dev installed in order to compile
go-blink1, btw.

For non-go-knowers like myself, you can say:

    cd $GOPATH
    mkdir src/github.com/gooseyard
    git clone https://github.com/GooseYArd/mutelight.git src/github.com/gooseyard/mutelight
    cd src/github.com/gooseyard/mutelight
    go get
    go build

I hit one snag in the form of a libusb issue with the mk3 device and
hink/go-blink1. A fix existed as a PR against hink's repo, but the repo seems
to be unloved, so I'm using todman's fork instead. Should todman's fork ever
disappear, I've stashed the patch in this repo as well- just apply it to the
go-blink1 sources before building.

I'll probably change the solid red light to the throbbing pattern that I was
generating with the Python bindings, but I was too lazy to figure it out with
go-blink(1) (although I think its probably trivial).

I'm not totally sure whether the default Pulse source is meant to have index 0
when listing sources, and I'm loathe to screw around with my Pulseaudio
configuration much since I've screwed it up many times before, so there's a
good chance you'll need to adjust the source index which is hardcoded in the
source.

Once its started, you can start it from an ~/.xsession file, or do what I did
and use a systemd user system unit.

Create the file ~/.config/systemd/user/mutelight.service

    [Unit]
    Description=MuteLight

    [Service]
    Type	    = simple
    Restart     = always
    RestartSec  = 10
    ExecStart   = /home/gooseyard/bin/mutelight

    [Install]
    WantedBy=session.target

Say:

systemctl --user daemon-reload
systemctl --user enable mutelight.service
systemctl --user start mutelight.service

To test, say:

    pactl set-source-mute 1 toggle


Using the index of your desired source. I use pavucontrol to verify that the
source is really muted when the light is on.



Happy hacking!
