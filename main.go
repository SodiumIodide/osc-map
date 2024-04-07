package main

import (
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/fsnotify/fsnotify"
	"github.com/hypebeast/go-osc/osc"
	log "github.com/sirupsen/logrus"

	"github.com/micmonay/keybd_event"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver

	yaml "gopkg.in/yaml.v3"
)

const (
	DefaultMidiIn          = "Keyboard"
	DefaultOSCOutIP        = "127.0.0.1"
	DefaultOSCOutPort      = 8765
	DefaultSampleRate      = 48000
	DefaultBufferSize      = 4800 // buffer size of 1/10 second
	DefaultResampleQuality = 4    // good balance of quality and playback time
)

type MSCMap struct {
	oscClient      *osc.Client
	midiOut        *drivers.Out
	qlabOut        *drivers.Out
	midiOutChannel uint8
	midiMap        map[float64]cueMap
	keyBonding     *keybd_event.KeyBonding
}

func main() {

	time.Sleep(5 * time.Second)
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
		log.Errorf("found options [%+v]", midi.GetInPorts())
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

	if conf.Outputs.KeyboardCommands {
		kb, err := keybd_event.NewKeyBonding()
		if err != nil {
			log.Errorf("failed to create key bonding: %v", err)
			quit = true
		} else {

			// For linux, it is very important to wait 2 seconds
			if runtime.GOOS == "linux" {
				log.Info("Please wait 2 seconds for keyboard binding...")
				time.Sleep(2 * time.Second)
			}

			mscMap.keyBonding = &kb
		}
	}

	if conf.Outputs.AudioFiles {
		speaker.Init(DefaultSampleRate, DefaultBufferSize)
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

			m.sendMidiOut(cue)
			if m.keyBonding != nil {
				m.sendKeyboardCommand(cue)
			}
			go m.playAudioFile(cue)
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

	return "", 0, fmt.Errorf("not an msc packet. len: %v bt[0]: %x", len(bt), bt[0])
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

// sendMidiPC sends a MIDI message to the midi out that configured in the config
func (m *MSCMap) sendMidiOut(cue float64) {
	if m.midiOut == nil {
		return
	}

	mc, ok := m.midiMap[cue]
	if !ok {
		log.Debugf("did not find cue mapping for cue[%v]", cue)
		return
	}

	soundCue := mc.soundCue
	muteCue := mc.muteCue
	unmuteCue := mc.unmuteCue
	faderCue := mc.faderCue
	faderVal := mc.faderVal

	if soundCue == 0 && len(muteCue) == 0 && len(unmuteCue) == 0 && len(faderCue) == 0 {
		return
	}

	if soundCue != 0 {
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
	}

	if len(muteCue) != 0 {
		for i := 0; i < len(muteCue); i++ {
			mm := midi.NoteOn(m.midiOutChannel, muteCue[i]-1, 0x7F)

			out, err := midi.SendTo(*m.midiOut)
			if err != nil {
				log.Errorf("failed to get midi send function: %v", err)
			}

			err = out(mm)
			if err != nil {
				log.Errorf("failed to send midi note message to [%v]: %v", m.midiOut, err)
				return
			}

			log.Infof("sent mute note %v to midi out", muteCue)
		}
	}

	if len(unmuteCue) != 0 {
		for i := 0; i < len(unmuteCue); i++ {
			mm := midi.NoteOn(m.midiOutChannel, unmuteCue[i]-1, 0x00)

			out, err := midi.SendTo(*m.midiOut)
			if err != nil {
				log.Errorf("failed to get midi send function: %v", err)
			}

			err = out(mm)
			if err != nil {
				log.Errorf("failed to send midi note message to [%v]: %v", m.midiOut, err)
				return
			}

			log.Infof("sent unmute note %v to midi out", muteCue)
		}
	}

	// Fader value can vary from 0 to 127, where 100 = U
	if len(faderCue) != 0 {
		if len(faderCue) != len(faderVal) {
			log.Errorf("each fader cue needs a fader value")
		}
		for i := 0; i < len(faderCue); i++ {
			if faderVal[i] > 127 {
				log.Errorf("fader value cannot be higher than 127")
			}

			mm := midi.ControlChange(m.midiOutChannel, faderCue[i]-1, faderVal[i])

			out, err := midi.SendTo(*m.midiOut)
			if err != nil {
				log.Errorf("failed to get midi send function: %v", err)
			}

			err = out(mm)
			if err != nil {
				log.Errorf("failed to send midi control change to [%v]: %v", m.midiOut, err)
				return
			}

			log.Infof("sent fader value %v, %v control change to midi out", faderCue, faderVal)
		}
	}

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

// sendKeyboardCommand simulates a keyboard keypress. Useful for soundboard programs
func (m *MSCMap) sendKeyboardCommand(cue float64) {
	if m.keyBonding == nil {
		log.Errorf("keybonding is nil")
		return
	}

	cueMap, ok := m.midiMap[cue]
	if !ok {
		log.Debugf("did not find cue mapping for cue[%v]", cue)
		return
	}

	if cueMap.keyboardKey == -1 {
		log.Debugf("no keyboard key specified for cue[%v]", cue)
		return
	}

	m.keyBonding.SetKeys(cueMap.keyboardKey)

	log.Debugf("sending keyboard: %v", cueMap.keyboardKey)

	// Press the selected keys
	err := m.keyBonding.Launching()
	if err != nil {
		log.Errorf("failed to launch key: %X", cueMap.keyboardKey)
	}

}

// play a simple audio file with no fading or level change
func (m *MSCMap) playAudioFile(cue float64) {
	mc, ok := m.midiMap[cue]
	if !ok {
		log.Debugf("did not find cue mapping for cue[%v]", cue)
		return
	}

	filename := mc.audioFile

	if filename == "" {
		log.Debugf("did not find audio file for cue[%v]", cue)
		return
	}

	fileExtension := filepath.Ext(filename)

	if fileExtension != ".mp3" && fileExtension != ".wav" {
		log.Errorf("incompatible file extension: %s", filename)
		return
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Errorf("cannot open file %s: %v", filename, err)
		return
	}

	if fileExtension == ".mp3" {
		streamer, format, err := mp3.Decode(file)
		if err != nil {
			log.Errorf("cannot decode file %s: %v", filename, err)
			return
		}
		defer streamer.Close()

		// buffer size of 1/10 of a second
		resampled := beep.Resample(DefaultResampleQuality, DefaultSampleRate, format.SampleRate, streamer)

		done := make(chan bool)
		speaker.Play(beep.Seq(resampled, beep.Callback(func() {
			done <- true
		})))

		<-done
	}

	if fileExtension == ".wav" {
		streamer, format, err := wav.Decode(file)
		if err != nil {
			log.Errorf("cannot decode file %s: %v", filename, err)
			return
		}
		defer streamer.Close()

		// buffer size of 1/10 of a second
		resampled := beep.Resample(DefaultResampleQuality, DefaultSampleRate, format.SampleRate, streamer)

		done := make(chan bool)
		speaker.Play(beep.Seq(resampled, beep.Callback(func() {
			done <- true
		})))

		<-done
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
	confBytes, err := os.ReadFile("config.yaml")
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
	midiMap := make(map[float64]cueMap)
	for _, cm := range conf.MidiCueMapping {

		// parse hex from config to int
		keyboard, ok := KeyboardMap[cm.Keyboard]
		if !ok {
			keyboard = -1
		}

		newCM := cueMap{
			soundCue:    cm.Sound,
			muteCue:     cm.Mute,
			unmuteCue:   cm.Unmute,
			faderCue:    cm.FaderChannel,
			faderVal:    cm.FaderValue,
			keyboardKey: keyboard,
			audioFile:   cm.AudioFile,
		}
		midiMap[cm.In] = newCM
	}

	m.midiMap = midiMap

	return conf, nil
}
