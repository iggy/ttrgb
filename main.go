package main

import (
	"bytes"
	"errors"
	"flag"
	"time"

	usb "github.com/google/gousb"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// These const's/var's adapted from:
// https://github.com/MoshiMoshi0/ttrgbplusapi
// This was a lot of help too:
// https://github.com/chestm007/linux_thermaltake_riing/
// https://godoc.org/github.com/google/gousb
// https://pkg.go.dev/mod/github.com/google/gousb
// https://github.com/google/gousb/blob/master/lsusb/main.go
// http://libusb.sourceforge.net/api-1.0/libusb_8h_source.html
// https://github.com/MoshiMoshi0/ttrgbplusapi
// https://github.com/chestm007/linux_thermaltake_riing
const (
	USBVID          = 0x264a
	USBPID          = 0x1fa5
	ProtocolGet     = 0x33
	ProtocolSet     = 0x32
	ProtocolFan     = 0x51
	ProtocolLight   = 0x52
	RGBSpeedExtreme = 0x00
	RGBSpeedFast    = 0x01
	RGBSpeedNormal  = 0x02
	RGBSpeedSlow    = 0x03
	RGBModeFlow     = 0x00 // +RGBSpeed
	RGBModeSpectrum = 0x04 // +RGBSpeed
	RGBModeRipple   = 0x08 // +RGBSpeed
	RGBModeBlink    = 0x0c // +RGBSpeed
	RGBModePulse    = 0x10 // +RGBSpeed
	RGBModeWave     = 0x14 // +RGBSpeed
	RGBModePerLed   = 0x18
	RGBModeFull     = 0x19
	ControllerPort1 = 0x01
	ControllerPort2 = 0x02
	ControllerPort3 = 0x03
	ControllerPort4 = 0x04
	ControllerPort5 = 0x05
)

var (
	StatusSuccess       = []byte{0xfc}
	StatusFail          = []byte{0xfe}
	CommandInit         = []byte{0xfe, 0x33} // returns StatusSuccess/Fail
	CommandGetFWVersion = []byte{0x33, 0x50} // returns Major,Minor,Patch
	CommandSaveProfile  = []byte{0x32, 0x53} // returns StatusSuccess/Fail
	CommandSetSpeed     = []byte{0x32, 0x51} // returns StatusSuccess/Fail
	CommandSetRGB       = []byte{0x32, 0x52} // returns StatusSuccess/Fail
	CommandGetData      = []byte{0x33, 0x51} // returns PORT, UNKNOWN, SPEED, RPM_L, RPM_H
)

var (
	verbose int
	red     int
	green   int
	blue    int
)

func init() {
	flag.IntVar(&verbose, "verbose", 3, "Verbose level (1-5)")
	flag.IntVar(&red, "red", 0, "red value for lights")
	flag.IntVar(&green, "green", 0, "green value for lights")
	flag.IntVar(&blue, "blue", 0, "blue value for lights")
}

func main() {
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.Level(5 % verbose)) // 5 = PanicLevel

	// Initialize a new Context.
	ctx := usb.NewContext()
	defer ctx.Close()

	// Open the FloeRiing controller
	dev, err := ctx.OpenDeviceWithVIDPID(USBVID, USBPID)
	if err != nil {
		log.Fatal().Err(err).Msg("Make sure the device under /dev/bus/usb is writable by your user/group")
	}
	if dev == nil {
		log.Fatal().Interface("dev", dev).Msg("device not found")
	}
	defer dev.Close()

	// the kernel binds a driver to the device by default, so we tell
	// libusb to detach it
	err = dev.SetAutoDetach(true)
	if err != nil {
		log.Fatal().Err(err).Msg("Error: SetAutoDetach:")
	}

	// Claim the default interface using a convenience function.
	// The default interface is always #0 alt #0 in the currently active
	// config.
	intf, done, err := dev.DefaultInterface()
	if err != nil {
		log.Fatal().Err(err).Msgf("Error: DefaultInterface(): %+v", dev)
	}
	defer done()
	log.Debug().Msgf("intf.String(): %+v", intf.String())

	// Open an OUT endpoint.
	oep, err := intf.OutEndpoint(1)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error: OutEndpoint(1): %+v", intf)
	}
	log.Debug().Msgf("oep.String: %+v", oep.String())

	// Open an IN endpoint
	iep, err := intf.InEndpoint(1)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error: InEndpoint(1): %+v (%+v)", intf)
	}
	log.Debug().Msgf("iep.String: %+v", iep.String())

	// send an init to the device, this is a step specific to this device
	// other devices need different bytes
	numBytes, err := oep.Write(CommandInit)
	if numBytes != len(CommandInit) {
		log.Fatal().Err(err).Msgf("Error Write(CommandInit): %+v - bytes written (), error %+v", oep.String(), numBytes)
	}
	log.Debug().Msg("init successfully sent to the endpoint")

	initRead := make([]byte, 64)
	numBytes, err = iep.Read(initRead)
	if numBytes != 64 || err != nil {
		log.Fatal().Err(err).Msgf("Error: failed to read CommandInit response: (%d) %+v", numBytes, initRead)
	}
	if !bytes.Equal(initRead[2:3], StatusSuccess) {
		log.Fatal().Msgf("Error: CommandInit failed: %+v", initRead[2:3])
	}
	log.Debug().Msgf("successful init: (%d) %+v", numBytes, initRead)

	// lightOn(oep, ControllerPort1, 255, 0, 0)
	lightOn(oep, ControllerPort1, 255, 255, 255)
	// time.Sleep(500 * time.Millisecond)
	lightOn(oep, ControllerPort2, 255, 125, 0)
	// lightOn(oep, ControllerPort2, 255, 255, 255)
	// time.Sleep(500 * time.Millisecond)
	// lightOn(oep, ControllerPort3, 255, 255, 0)
	// lightOn(oep, ControllerPort3, 255, 255, 255)
	// time.Sleep(500 * time.Millisecond)
	// lightOn(oep, ControllerPort4, 0, 255, 255)
	// lightOn(oep, ControllerPort4, 255, 255, 255)
	// time.Sleep(500 * time.Millisecond)
	// lightPulse(oep, ControllerPort1, RGBSpeedSlow, 255, 0, 0)

	time.Sleep(5 * time.Second)

	// setSpeed(oep, iep, ControllerPort1, 20)
	// setSpeed(oep, iep, ControllerPort2, 20)
	// setSpeed(oep, iep, ControllerPort3, 20)
	// setSpeed(oep, iep, ControllerPort4, 20)

	// time.Sleep(5 * time.Second)
	lightOff(oep, ControllerPort1)
	lightOff(oep, ControllerPort2)
	// time.Sleep(500 * time.Millisecond)
	// time.Sleep(500 * time.Millisecond)
	lightOff(oep, ControllerPort3)
	// time.Sleep(500 * time.Millisecond)
	// lightOff(oep, ControllerPort4)

}

// turn on a specific port to full color
func lightOn(ep *usb.OutEndpoint, port byte, r byte, g byte, b byte) error {
	lightOn := append(CommandSetRGB, port, RGBModeFull, g, r, b)
	numBytes, err := ep.Write(lightOn)
	if numBytes != len(lightOn) {
		log.Fatal().Err(err).Msgf("ErrorWrite(lightOn): %+v: bytes written (%s), returned error", ep.String(), numBytes)
	}
	log.Debug().Msg("lightOn successfully sent to the endpoint")

	return nil
}

func lightOff(ep *usb.OutEndpoint, port byte) error {
	lightOff := append(CommandSetRGB, port, RGBModeFull, 0, 0, 0)
	numBytes, err := ep.Write(lightOff)
	if numBytes != len(lightOff) {
		log.Fatal().Err(err).Msgf("ErrorWrite(lightOff): %+v: bytes written (%s), returned error", ep.String(), numBytes)
	}
	log.Debug().Msg("lightOff successfully sent to the endpoint")

	return nil
}

func lightPulse(ep *usb.OutEndpoint, port byte, speed byte, r byte, g byte, b byte) error {
	lightPulse := append(CommandSetRGB, port, RGBModePulse+speed, g, r, b)
	numBytes, err := ep.Write(lightPulse)
	if numBytes != len(lightPulse) {
		log.Fatal().Err(err).Msgf("ErrorWrite(lightPulse): %+v: bytes written (%s), returned error", ep.String(), numBytes)
	}
	log.Debug().Msg("lightPulse successfully sent to the endpoint")
	return nil
}

func setSpeed(oep *usb.OutEndpoint, iep *usb.InEndpoint, port byte, speed byte) error {
	command := append(CommandSetSpeed, port, 0x01, speed)
	numBytes, err := oep.Write(command)
	if numBytes != len(command) {
		log.Fatal().Err(err).Msgf("ErrorWrite(lightOff): bytes written (%s), returned error", numBytes)
	}
	log.Debug().Msg("setSpeed successfully sent to the endpoint")

	commandRead := make([]byte, 64)
	numBytes, err = iep.Read(commandRead)
	if numBytes != 64 || err != nil {
		log.Error().Err(err).Msgf("Error: failed to read command response %s %s", numBytes, commandRead)
		return errors.New("failed to read from setSpeed command")
	}
	if !bytes.Equal(commandRead[2:3], StatusSuccess) {
		log.Error().Err(err).Msgf("Error: Command failed: %s", commandRead[2:3])
		return errors.New("device failed to process setSpeed command")
	}
	log.Debug().Msgf("successful init: %s %s", numBytes, commandRead)

	return nil
}
