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
	HomeAssistant    bool             `yaml:"homeassistant"`
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
	In           float64  `yaml:"light"`
	Sound        uint8    `yaml:"sound"`
	Mute         []uint8  `yaml:"mute"`
	Unmute       []uint8  `yaml:"unmute"`
	FaderChannel []uint8  `yaml:"fader"`
	FaderValue   []uint8  `yaml:"value"`
	Keyboard     string   `yaml:"keyboard"`
	AudioFile    string   `yaml:"file"`
	HouseLight   []uint8  `yaml:"houselight"`
	RGBW         []uint8  `yaml:"rgbw"`
	Transitions  []uint8  `yaml:"transitions"`
	Effects      []string `yaml:"effects"`
}

type cueMap struct {
	soundCue    uint8
	muteCue     []uint8
	unmuteCue   []uint8
	faderCue    []uint8
	faderVal    []uint8
	keyboardKey int
	audioFile   string
	houseLight  []uint8
	rgbw        []uint8
	transitions []uint8
	effects     []string
}

// Struct to represent the HomeAssistant API response
type Response struct {
	State string `json:"state"`
}

// Struct for light control data
type LightRequestData struct {
	entity_id  string  `json:"entity_id"`
	rgbw_color []uint8 `json:"effect"`
	transition uint8   `json:"transition"`
	effect     string  `json:"effect"`
}
