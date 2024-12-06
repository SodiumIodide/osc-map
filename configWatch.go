package main

import (
	"os"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

// monitorConfig watches for changes in the config and will update the midiMap in real time so the program doesn't need to be restarted when a new cue is added to the config
func (m *OSCMap) monitorConfig() {
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

func (m *OSCMap) readConfig() (*conf, error) {
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
	controlMap := make(map[string]cueMap)
	for _, cm := range conf.ControlCueMapping {

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
			houseLights: cm.HouseLights,
			rgbws:       cm.RGBWs,
			transitions: cm.Transitions,
			effects:     cm.Effects,
		}
		controlMap[cm.In] = newCM
	}

	m.controlMap = controlMap

	return conf, nil
}
