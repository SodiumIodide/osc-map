# msc-osc

Receives MSC (MIDI Show Control) messages from an etc express (sysex messages), sends it out as an osc message, and sends out a midi program change message. I made ths just to control scenes in QLC+ with the etc express. Tested on Windows and MacOS.


## Dependencies
- requires CGO

## Build
`go build`

## Config
msc-osc will look for `config.yaml` in the local directory

| Key                     | Value Type | Description                                                                                                |
|-------------------------|------------|------------------------------------------------------------------------------------------------------------|
| midiIn                  | string     | name of the midi port that you want to receive input from                                                  |
| outputs.osc.ip          | ip address | the ip address to send osc messages to                                                                     |
| outputs.osc.port        | int        | the port of to send osc messages to                                                                        |
| outputs.midi-pc.name    | string     | name of the midi port that you want to send program change messages to                                     |
| outputs.midi-pc.channel | int        | the midi channel that you want to send program change messages to                                          |
| outputs.qlab            | boolean    | true or false depending on if you want to send program change messages to qlab running on the same machine |
| midi-cue-mapping        | array      | list of midi cue mappings                                                                                  |
| midi-cue-mapping.light  | double     | the light cue to listen for from the etc express light board                                               |
| midi-cue-mapping.sound  | int        | the program change cue to send to the tt24 sound board                                                     |
## output msc message format:
```
address = /msc/<command>/<cue number>
message = <cue number> true <command>
```

## example packets:
midi input: `F0 7F 01 02 01 01 32 36 35 00 31 00 F7`  (go cue 265 A/B fader)

osc output: `/msc/go/265 ,iTs 265 true go`