package main

import "net"

type conf struct {
	MidiIn string `yaml:"midiIn"`

	Outputs        confOutputs  `yaml:"outputs"`
	MidiCueMapping []cueMapping `yaml:"midi-cue-mapping"`
}

type confOutputs struct {
	OSC    confOutputOSC    `yaml:"osc"`
	MIDIPC confOutputMIDIPC `yaml:"midi-pc"`
}

type confOutputOSC struct {
	IP   net.IP `yaml:"ip"`
	Port int    `yaml:"port"`
}

type confOutputMIDIPC struct {
	Name    string `yaml:"name"`
	Channel uint8  `yaml:"channel"`
}

type cueMapping struct {
	In  float64 `yaml:"light"`
	Out uint8   `yaml:"sound"`
}
