package main

import "net"

type conf struct {
	OSCIn confOSC `yaml:"oscIn"`

	Outputs           confOutputs      `yaml:"outputs"`
	ControlCueMapping []confCueMapping `yaml:"control-cue-mapping"`
}

type confOutputs struct {
	OSCOut           confOSC          `yaml:"oscOut"`
	MIDIPC           confOutputMIDIPC `yaml:"midi-pc"`
	MIDISCS          confOutputMIDIPC `yaml:"midi-scs"`
	Qlab             bool             `yaml:"qlab"`
	KeyboardCommands bool             `yaml:"keyboard-commands"`
	AudioFiles       bool             `yaml:"audio-files"`
}

type confOSC struct {
	IP   net.IP `yaml:"ip"`
	Port int    `yaml:"port"`
}

type confOutputMIDIPC struct {
	Name    string `yaml:"name"`
	Channel uint8  `yaml:"channel"`
}

type confCueMapping struct {
	In           string    `yaml:"light"`
	Sound        uint8     `yaml:"sound"`
	Mute         []uint8   `yaml:"mute"`
	Unmute       []uint8   `yaml:"unmute"`
	FaderChannel []uint8   `yaml:"fader"`
	FaderValue   []uint8   `yaml:"value"`
	Keyboard     string    `yaml:"keyboard"`
	AudioFile    string    `yaml:"file"`
	HouseLights  []int     `yaml:"houselights"`
	RGBWs        [][]int   `yaml:"rgbws"`
	Transitions  []float32 `yaml:"transitions"`
	Effects      []string  `yaml:"effects"`
	SCS          uint8     `yaml:"scs"`
}

type cueMap struct {
	soundCue    uint8
	muteCue     []uint8
	unmuteCue   []uint8
	faderCue    []uint8
	faderVal    []uint8
	keyboardKey int
	audioFile   string
	houseLights []int
	rgbws       [][]int
	transitions []float32
	effects     []string
	scs         uint8
}

// Struct to represent the HomeAssistant API response
type Response struct {
	State string `json:"state"`
}

// Struct for light control data
type LightRequestData struct {
	Entity_id  string  `json:"entity_id"`
	Rgbw_color []int   `json:"rgbw_color"`
	Transition float32 `json:"transition"`
	Effect     string  `json:"effect"`
}
