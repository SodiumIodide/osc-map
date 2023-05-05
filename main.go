package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"strconv"

	"github.com/fsnotify/fsnotify"
	"github.com/hypebeast/go-osc/osc"
	log "github.com/sirupsen/logrus"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver

	yaml "gopkg.in/yaml.v3"
)

const (
	DefaultMidiIn     = "Keyboard"
	DefaultOSCOutIP   = "127.0.0.1"
	DefaultOSCOutPort = 8765
)

type MSCMap struct {
	oscClient      *osc.Client
	midiOut        *drivers.Out
	qlabOut        *drivers.Out
	midiOutChannel uint8
	midiMap        map[float64]uint8
}

func main() {
	defer midi.CloseDriver()

	log.SetLevel(log.DebugLevel)

	mscMap := &MSCMap{}
	conf, err := mscMap.readConfig()
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}
	go mscMap.monitorConfig()

	log.Debugf("final midi mapping: %v", mscMap.midiMap)

	// setup osc client
	mscMap.oscClient = osc.NewClient(conf.Outputs.OSC.IP.String(), conf.Outputs.OSC.Port)

	quit := false
	// connect to midi input
	in, err := midi.FindInPort(conf.MidiIn)
	if err != nil {
		log.Errorf("can't find midi input %v", conf.MidiIn)
		quit = true
	}

	// connect to midi output
	out, err := midi.FindOutPort(conf.Outputs.MIDIPC.Name)
	if err != nil {
		log.Errorf("can't find midi output %v", conf.Outputs.MIDIPC.Name)
		quit = true
	} else {
		mscMap.midiOut = &out
	}

	// connect to qlab if we're using that
	if conf.Outputs.Qlab {
		out, err := midi.FindOutPort("QLab")
		if err != nil {
			log.Errorf("can't find midi output %v", "QLab")
			quit = true
		} else {
			mscMap.qlabOut = &out
		}
	}

	if quit {
		return
	}

	// listen for midi sysex commands from etc
	stop, err := midi.ListenTo(in, mscMap.midiListenFunc, midi.UseSysEx())
	if err != nil {
		log.Errorf("failed to listen to midi: %v", err)
		return
	}

	log.Infof("listening for midi from %v(%v) and outputting to %s:%d and %s", in.String(), in.Number(), conf.Outputs.OSC.IP, conf.Outputs.OSC.Port, conf.Outputs.MIDIPC.Name)

	// listen for ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for range c {
		// sig is a ^C, handle it
		fmt.Println("quitting")
		break
	}

	stop()
}

// midiListenFunc listens for messages coming from the etc express, parses it, and sends it out to the output midi and as an osc command
func (m *MSCMap) midiListenFunc(msg midi.Message, timestampms int32) {
	var bt []byte
	var ch, key, vel uint8
	switch {
	case msg.GetSysEx(&bt):
		log.Debugf("got sysex: % X", bt)
		command, cue, err := parseMSC(bt)
		if err != nil {
			log.Errorf("failed to parse msc: %v", err)
		} else {
			tc := fmt.Sprintf("%.1f", cue)
			if string(tc[len(tc)-1:]) == "0" {
				tc = fmt.Sprintf("%.0f", cue)
			}

			m.sendMidiPC(cue)
			m.sendOSC(command, tc)
		}
	case msg.GetNoteStart(&ch, &key, &vel):
		log.Debugf("got starting note %s on channel %v with velocity %v", midi.Note(key), ch, vel)
	case msg.GetNoteEnd(&ch, &key):
		log.Debugf("got ending note %s on channel %v", midi.Note(key), ch)
	default:
		// ignore
	}
}

// parseMSC will parse the etc msc message into a command string and a cue number
func parseMSC(bt []byte) (command string, cue float64, err error) {
	if len(bt) >= 9 && bt[0] == 0x7f {
		// get cue number
		btLen := len(bt)
		cue, err := strconv.ParseFloat(string(bt[5:btLen-3]), 64)
		if err != nil {
			return "", 0, fmt.Errorf("failed to parse float from[%v]: %v", string(bt[5:btLen-3]), err)
		}

		// get command
		command = ""
		switch bt[4] {
		case 0x01:
			command = "go"
		case 0x02:
			command = "stop"
		case 0x03:
			command = "resume"
		case 0x07:
			command = "macro"
		default:
			return "", 0, fmt.Errorf("unrecognized msc command: %x", bt[4])
		}

		return command, cue, nil
	}

	return "", 0, fmt.Errorf("not an msc packet. len: %v bt[0]: %x\n", len(bt), bt[0])
}

// sendOSC sends a message out as an osc message with address /msc/<command>/<cue number>
func (m *MSCMap) sendOSC(command, cue string) {
	cueFloat, err := strconv.ParseFloat(cue, 64)
	if err != nil {
		log.Errorf("failed to convert %v to int: %v", cue, err)
	} else {
		msg := osc.NewMessage(fmt.Sprintf("/msc/%s/%s", command, cue))
		msg.Append(cueFloat)
		msg.Append(command)
		log.Infof("sending osc %v\n", msg.String())
		m.oscClient.Send(msg)
	}
}

// sendMidiPC sends a program change message to the midi out that configured in the config
func (m *MSCMap) sendMidiPC(cue float64) {
	if m.midiOut == nil {
		return
	}

	soundCue, ok := m.midiMap[cue]
	if !ok {
		log.Debugf("did not find cue mapping for cue[%v]", cue)
		return
	}

	mm := midi.ProgramChange(m.midiOutChannel, soundCue-1)

	out, err := midi.SendTo(*m.midiOut)
	if err != nil {
		log.Errorf("failed to get midi send function: %v", err)
	}

	err = out(mm)
	if err != nil {
		log.Errorf("failed to send midi program change message to [%v]: %v", m.midiOut, err)
		return
	}

	log.Infof("sent program change %v to midi out", soundCue)

	if m.qlabOut != nil {
		mm := midi.ProgramChange(m.midiOutChannel, soundCue)

		out, err := midi.SendTo(*m.qlabOut)
		if err != nil {
			log.Errorf("failed to get midi send function: %v", err)
		}

		err = out(mm)
		if err != nil {
			log.Errorf("failed to send midi program change message to [%v]: %v", m.qlabOut, err)
			return
		}

		log.Infof("sent program change %v to qlab", soundCue)
	}
}

// sendAll is only for testing what messages qlc+ can see
func (m *MSCMap) sendAll() {
	x := big.NewRat(1, 10)
	y := big.NewRat(9999, 10)
	z := big.NewRat(1, 10)
	for i := x; i.Cmp(y) <= 0; i = i.Add(i, z) {
		f, _ := i.Float64()
		fmt.Println(f)
		fs := fmt.Sprintf("%.1f", f)
		if string(fs[len(fs)-1:]) == "0" {
			m.sendOSC("go", fmt.Sprintf("%.0f", f))
		}
		m.sendOSC("go", fmt.Sprintf("%.1f", f))
	}
}

// monitorConfig watches for changes in the config and will update the midiMap in real time so the program doesn't need to be restarted when a new cue is added to the config
func (m *MSCMap) monitorConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("NewWatcher failed: ", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				_, err := m.readConfig()
				if err != nil {
					log.Errorf("failed to read config: %v", err)
				}

				log.Infof("config file changed: %s %s", event.Name, event.Op)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}

	}()

	err = watcher.Add("config.yaml")
	if err != nil {
		log.Fatal("Add failed:", err)
	}
	<-done
}

func (m *MSCMap) readConfig() (*conf, error) {
	confBytes, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	conf := &conf{}
	err = yaml.Unmarshal(confBytes, conf)
	if err != nil {
		log.Fatalf("failed to unmarshal config file: %v", err)
	}

	// print config and exit
	log.Debugf("config: %+v", conf)

	// create midi map
	midiMap := make(map[float64]uint8)
	for _, cm := range conf.MidiCueMapping {
		midiMap[cm.In] = cm.Out
	}

	m.midiMap = midiMap

	return conf, nil
}
