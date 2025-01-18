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
	NumHouseLights           = 15
)

type OSCMap struct {
	oscDispatcher  *osc.StandardDispatcher
	oscInServer    *osc.Server
	oscOutClient   *osc.Client
	midiOut        *drivers.Out
	scsOut         *drivers.Out
	qlabOut        *drivers.Out
	midiOutChannel uint8
	scsOutChannel  uint8
	controlMap     map[string]cueMap
	keyBonding     *keybd_event.KeyBonding
}

// There are 15 house lights and each needs a stop channel for custom effects
var stopChannels = make([]chan struct{}, NumHouseLights)

func listenForOSC(m *OSCMap, responseChannel chan bool) {
	m.oscDispatcher.AddMsgHandler("/cs/out/ping", func(msg *osc.Message) {
		// Check ping response
		responseChannel <- true
	})

	// ExtractDecimal extracts the decimal part from a string in the format "decimal_label"
	ExtractDecimal := func(input string) string {
		// Find the position of the underscore
		underscoreIndex := strings.Index(input, "_")
		if underscoreIndex == -1 {
			// If no underscore is found, return the entire string (assuming it's just the decimal)
			return input
		}
		// Return the substring before the underscore
		return input[:underscoreIndex]
	}

	// Handle cue numbers
	m.oscDispatcher.AddMsgHandler("/cs/out/playback/go", func(msg *osc.Message) {
		cueNumber := fmt.Sprintf("%v", msg.Arguments[0])

		// Trim the cue label and underscore
		cueNumber = ExtractDecimal(cueNumber)

		// If cue number ends in 0, make an optional second to test
		cueInteger := strings.Clone(cueNumber)
		if strings.Contains(cueInteger, ".0") {
			cueInteger = strings.ReplaceAll(cueInteger, ".0", "")
		}
		log.Infof("Received cue number: %v", cueNumber)

		go m.sendMidiOut(cueNumber, cueInteger)
		if m.keyBonding != nil {
			go m.sendKeyboardCommand(cueNumber, cueInteger)
		}
		go m.playAudioFile(cueNumber, cueInteger)
		go m.toggleLight(cueNumber, cueInteger)
	})

	err := m.oscInServer.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting OSC server: %v", err)
	}
}

func main() {
	time.Sleep(5 * time.Second)
	defer midi.CloseDriver()

	log.SetLevel(log.DebugLevel)

	oscMap := &OSCMap{}
	conf, err := oscMap.readConfig()
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	go oscMap.monitorConfig()

	log.Debugf("Final cue mapping: %v", oscMap.controlMap)

	// set up osc dispatcher and server
	oscMap.oscDispatcher = osc.NewStandardDispatcher()
	oscMap.oscInServer = &osc.Server{
		Addr:       fmt.Sprint(conf.OSCIn.IP.String(), ":", conf.OSCIn.Port),
		Dispatcher: oscMap.oscDispatcher,
	}
	defer oscMap.oscInServer.CloseConnection()

	// set up osc send client
	oscMap.oscOutClient = osc.NewClient(conf.Outputs.OSCOut.IP.String(), conf.Outputs.OSCOut.Port)

	quit := false

	// connect to midi output
	out, err := midi.FindOutPort(conf.Outputs.MIDIPC.Name)
	if err != nil {
		log.Errorf("Can't find midi output %v", conf.Outputs.MIDIPC.Name)
		quit = true
	} else {
		oscMap.midiOut = &out
		oscMap.midiOutChannel = conf.Outputs.MIDIPC.Channel
	}

	scs, err := midi.FindOutPort(conf.Outputs.MIDISCS.Name)
	if err != nil {
		log.Errorf("Can't find midi output %v", conf.Outputs.MIDISCS.Name)
		quit = true
	} else {
		oscMap.scsOut = &scs
		oscMap.scsOutChannel = conf.Outputs.MIDISCS.Channel
	}

	// connect to qlab if we're using that
	if conf.Outputs.Qlab {
		out, err := midi.FindOutPort("QLab")
		if err != nil {
			log.Errorf("Can't find midi output %v", "QLab")
			quit = true
		} else {
			oscMap.qlabOut = &out
		}
	}

	for i := 0; i < NumHouseLights; i++ {
		stopChannels[i] = make(chan struct{})
	}

	if conf.Outputs.AudioFiles {
		speaker.Init(DefaultSampleRate, DefaultBufferSize)
	}

	if conf.Outputs.KeyboardCommands {
		kb, err := keybd_event.NewKeyBonding()
		if err != nil {
			log.Errorf("Failed to create key bonding: %v", err)
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
		log.Infof("Successfully received a ping response from %v:%v", conf.Outputs.OSCOut.IP.String(), conf.Outputs.OSCOut.Port)
		timer.Stop()
	case <-timer.C:
		log.Fatalf("No ping response detected from %v:%v", conf.Outputs.OSCOut.IP.String(), conf.Outputs.OSCOut.Port)
		return
	}

	log.Infof("Listening for OSC from %v:%v, outputting OSC to %s:%d and MIDI to %s", conf.OSCIn.IP, conf.OSCIn.Port, conf.Outputs.OSCOut.IP, conf.Outputs.OSCOut.Port, conf.Outputs.MIDIPC.Name)

	// listen for ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for range c {
		// sig is a ^C, handle it
		fmt.Println("Quitting")
		break
	}
}
