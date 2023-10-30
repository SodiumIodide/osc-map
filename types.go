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
	In       float64 `yaml:"light"`
	Sound    uint8   `yaml:"sound"`
	Keyboard string  `yaml:"keyboard"`
}

type cueMap struct {
	soundCue    uint8
	keyboardKey int
}
