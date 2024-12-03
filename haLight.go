package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func sendRequestJSON(lightID int, rgbw []int, transition float32, effect string) {
	url := fmt.Sprintf("%s:%d/api/services/light/turn_on", DefaultHomeAssistantHTTP, DefaultHomeAssistantPort)

	data := LightRequestData{
		Entity_id:  fmt.Sprintf("light.house_light_%d", lightID),
		Rgbw_color: rgbw,
		Transition: transition,
		Effect:     effect,
	}

	// Make client
	client := &http.Client{}

	jsonData, err := json.Marshal(&data)
	if err != nil {
		log.Errorf("unable to create json data: %v", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("HAKEY")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error sending request: %v", err)
	}

	defer resp.Body.Close()
	//fmt.Println("Response Status:", resp.Status)
}

// Define a function meant to be edited/rebuilt for timing and debug
func customRainbow(lightID int, transition float32, sleep float32, stopChannel <-chan struct{}) {
	for {
		select {
		case <-stopChannel:
			return
		default:
			state := rand.IntN(11)
			switch state {
			case 0:
				sendRequestJSON(lightID, []int{255, 0, 0, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 1:
				sendRequestJSON(lightID, []int{255, 128, 0, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 2:
				sendRequestJSON(lightID, []int{255, 255, 0, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 3:
				sendRequestJSON(lightID, []int{128, 255, 0, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 4:
				sendRequestJSON(lightID, []int{0, 255, 0, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 5:
				sendRequestJSON(lightID, []int{0, 255, 128, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 6:
				sendRequestJSON(lightID, []int{0, 255, 255, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 7:
				sendRequestJSON(lightID, []int{0, 128, 255, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 8:
				sendRequestJSON(lightID, []int{0, 0, 255, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 9:
				sendRequestJSON(lightID, []int{128, 0, 255, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 10:
				sendRequestJSON(lightID, []int{255, 0, 255, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state++
			case 11:
				sendRequestJSON(lightID, []int{255, 0, 128, 0}, transition, "None")
				time.Sleep(time.Duration(sleep) * time.Second)
				state = 0
			}
		}
	}
}

func (m *MSCMap) toggleLight(cue float64) {
	mc, ok := m.midiMap[cue]
	if !ok {
		log.Debugf("did not find cue mapping for cue[%v]", cue)
		return
	}

	lightIDs := mc.houseLights
	transitions := mc.transitions
	effects := mc.effects
	rgbws := mc.rgbws

	if len(lightIDs) != 0 {
		// Check length errors
		if len(lightIDs) != len(transitions) {
			if len(transitions) != 1 {
				log.Errorf("unmatched transitions list length to number of lights in cue[%v]", cue)
				return
			}
		}
		if len(lightIDs) != len(effects) {
			if len(effects) != 1 {
				log.Errorf("unmatched effects list length to number of lights in cue[%v]", cue)
				return
			}
		}
		if len(lightIDs) != len(rgbws) {
			if len(rgbws) != 1 {
				log.Errorf("unmatched RGBWs list length to number of lights in cue[%v]", cue)
			}
		}

		for i := 0; i < len(lightIDs); i++ {
			log.Debugf("Sending light cue to house light %v", lightIDs[i])

			// Prepare the HTTP request
			var effect string
			if len(effects) == 1 {
				effect = effects[0]
			} else {
				effect = effects[i]
			}
			var transition float32
			if len(transitions) == 1 {
				transition = transitions[0]
			} else {
				transition = transitions[i]
			}
			var rgbw []int
			if len(rgbws) == 1 {
				rgbw = rgbws[0]
			} else {
				rgbw = rgbws[i]
			}

			// Error check RGBW
			for color := 0; color < 4; color++ {
				if color < 0 || color > 255 {
					log.Errorf("Invalid RGBW: %v", color)
				}
			}

			lightID := lightIDs[i]
			sendRequest := func(lightID int, transition float32, effect string, rgbw []int) {
				// Check effect type - important to set for transition times to or away from light board control
				if effect == "None" {
					close(stopChannels[lightID-1])
					stopChannels[lightID-1] = make(chan struct{})

					sendRequestJSON(lightID,
						[]int{0, 0, 0, 0},
						0,
						"None")

					sendRequestJSON(lightID,
						rgbw,
						transition,
						"None")
				} else if effect == "Light Board Control" {
					close(stopChannels[lightID-1])
					stopChannels[lightID-1] = make(chan struct{})

					sendRequestJSON(lightID,
						rgbw,
						transition,
						"None")

					time.Sleep(time.Duration(transition) * time.Second)

					sendRequestJSON(lightID,
						rgbw,
						0,
						"Light Board Control")
				} else if effect == "Custom Rainbow" {
					sendRequestJSON(lightID,
						[]int{0, 0, 0, 0},
						0,
						"None")

					// INFO: Edit these arguments to alter custom rainbow - requires recompiling
					// lightID, transition, sleep
					go customRainbow(lightID, transition-0.1, transition+0.1, stopChannels[lightID-1])
				} else {
					close(stopChannels[lightID-1])
					stopChannels[lightID-1] = make(chan struct{})

					sendRequestJSON(lightID,
						[]int{0, 0, 0, 0},
						0,
						"None")

					sendRequestJSON(lightID,
						rgbw,
						transition,
						effect)
				}
			}

			go sendRequest(lightID, transition, effect, rgbw)
		}
	}
}
