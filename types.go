package main

import "net"

type conf struct {
	MidiIn string `yaml:"midiIn"`

	Outputs        confOutputs      `yaml:"outputs"`
	MidiCueMapping []confCueMapping `yaml:"midi-cue-mapping"`
}

type confOutputs struct {
	OSC              confOutputOSC    `yaml:"osc"`
	MIDIPC           confOutputMIDIPC `yaml:"midi-pc"`
	Qlab             bool             `yaml:"qlab"`
	KeyboardCommands bool             `yaml:"keyboard-commands"`
	AudioFiles       bool             `yaml:"audio-files"`
}

type confOutputOSC struct {
	IP   net.IP `yaml:"ip"`
	Port int    `yaml:"port"`
}

type confOutputMIDIPC struct {
	Name    string `yaml:"name"`
	Channel uint8  `yaml:"channel"`
}

type confCueMapping struct {
	In           float64   `yaml:"light"`
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
