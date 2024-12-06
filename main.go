package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/faiface/beep/speaker"
	"github.com/hypebeast/go-osc/osc"
	log "github.com/sirupsen/logrus"

	"github.com/micmonay/keybd_event"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

const (
	DefaultMidiIn            = "Keyboard"
	DefaultOSCInIP           = "10.1.10.203"
	DefaultOSCInPort         = 8006
	DefaultOSCOutIP          = "10.1.10.77"
	DefaultOSCListenPort     = 8005
	DefaultSampleRate        = 48000
	DefaultBufferSize        = 4800 // buffer size of 1/10 second
	DefaultResampleQuality   = 4    // good balance of quality and playback time
	DefaultHomeAssistantHTTP = "http://homeassistant.local"
	DefaultHomeAssistantPort = 80
	numHouseLights           = 15
)

type OSCMap struct {
	oscDispatcher  *osc.StandardDispatcher
	oscInServer    *osc.Server
	oscOutClient   *osc.Client
	midiOut        *drivers.Out
	qlabOut        *drivers.Out
	midiOutChannel uint8
	controlMap     map[string]cueMap
	keyBonding     *keybd_event.KeyBonding
}

// There are 15 house lights and each needs a stop channel for custom effects
var stopChannels = make([]chan struct{}, numHouseLights)

func main() {
	time.Sleep(5 * time.Second)
	defer midi.CloseDriver()

	log.SetLevel(log.DebugLevel)

	oscMap := &OSCMap{}
	conf, err := oscMap.readConfig()
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}
	go oscMap.monitorConfig()

	log.Debugf("final cue mapping: %v", oscMap.controlMap)

	quit := false

	// setup osc dispatcher and server
	oscMap.oscDispatcher = osc.NewStandardDispatcher()
	oscMap.oscInServer = &osc.Server{
		Addr:       fmt.Sprint(conf.OSCIn.IP.String(), ":", conf.OSCIn.Port),
		Dispatcher: oscMap.oscDispatcher,
	}

	// set up osc send client
	oscMap.oscOutClient = osc.NewClient(conf.Outputs.OSCOut.IP.String(), conf.Outputs.OSCOut.Port)

	// connect to midi output
	out, err := midi.FindOutPort(conf.Outputs.MIDIPC.Name)
	if err != nil {
		log.Errorf("can't find midi output %v", conf.Outputs.MIDIPC.Name)
		quit = true
	} else {
		oscMap.midiOut = &out
	}

	// connect to qlab if we're using that
	if conf.Outputs.Qlab {
		out, err := midi.FindOutPort("QLab")
		if err != nil {
			log.Errorf("can't find midi output %v", "QLab")
			quit = true
		} else {
			oscMap.qlabOut = &out
		}
	}

	for i := 0; i < numHouseLights; i++ {
		stopChannels[i] = make(chan struct{})
	}

	if conf.Outputs.AudioFiles {
		speaker.Init(DefaultSampleRate, DefaultBufferSize)
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

			oscMap.keyBonding = &kb
		}
	}

	if quit {
		return
	}

	// Ping the Colorsource AV to open a loopback, then listen for cue numbers
	responseChannel := make(chan bool, 1)
	go listenForOSC(oscMap, responseChannel)

	pingMessage := osc.NewMessage("/cs/ping", "1")
	if err := oscMap.oscOutClient.Send(pingMessage); err != nil {
		log.Errorf("Failed to ping: %v", err)
	}

	// Wait for a response or a timeout
	timer := time.NewTimer(time.Second * 5)
	select {
	case <-responseChannel:
		//Received a response
		log.Debugf("Successfully sent a ping")
		timer.Stop()
	case <-timer.C:
		log.Errorf("No ping response detected")
		return
	}

	log.Infof("listening for OSC from %v:%v and outputting MIDI to %s:%d and %s", conf.OSCIn.IP, conf.OSCIn.Port, conf.Outputs.OSCOut.IP, conf.Outputs.OSCOut.Port, conf.Outputs.MIDIPC.Name)

	// listen for ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for range c {
		// sig is a ^C, handle it
		fmt.Println("quitting")
		break
	}
}

func listenForOSC(m *OSCMap, responseChannel chan bool) {
	m.oscDispatcher.AddMsgHandler("/cs/out/ping", func(msg *osc.Message) {
		// Check ping response
		responseChannel <- true
	})

	// Handle cue numbers
	m.oscDispatcher.AddMsgHandler("/cs/out/playback/go", func(msg *osc.Message) {
		cueNumber := fmt.Sprintf("%v", msg.Arguments[0])

		// Trim one trailing '_'
		if last := len(cueNumber) - 1; last >= 0 && cueNumber[last] == '_' {
			cueNumber = cueNumber[:last]
		}

		// If cue number ends in 0, make an optional second to test
		cueInteger := strings.Clone(cueNumber)
		if strings.Contains(cueInteger, ".0") {
			cueInteger = strings.ReplaceAll(cueInteger, ".0", "")
		}
		log.Debugf("Received cue number: %v", cueNumber)

		m.sendMidiOut(cueNumber, cueInteger)
		if m.keyBonding != nil {
			m.sendKeyboardCommand(cueNumber, cueInteger)
		}
		go m.playAudioFile(cueNumber, cueInteger)
		go m.toggleLight(cueNumber, cueInteger)
	})

	err := m.oscInServer.ListenAndServe()
	if err != nil {
		log.Errorf("Error starting OSC server: %v", err)
	}
}

// sendMidiOut sends a MIDI message to the midi out that configured in the config
func (m *OSCMap) sendMidiOut(cueNumber string, cueInteger string) {
	if m.midiOut == nil {
		return
	}

	mc, ok := m.controlMap[cueNumber]
	if !ok {
		mc, ok = m.controlMap[cueInteger]
		if !ok {
			log.Debugf("no soundboard interface command for cue[%v]", cueNumber)
			return
		}
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
			return
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
		}

		log.Infof("sent mute note %v to midi out", muteCue)
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
		}

		log.Infof("sent unmute note %v to midi out", unmuteCue)
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
		}

		log.Infof("sent fader value %v, %v control change to midi out", faderCue, faderVal)
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
