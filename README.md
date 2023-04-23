# msc-osc

Receives MSC (MIDI Show Control) messages from an etc express (sysex messages) and sends it out as an osc message. I made ths just to control scenes in QLC+ with the etc express. Tested on Windows and MacOS.


## Dependencies
- requires CGO

## Build
`go build`

## Config
msc-osc will look for `config.yaml` in the local directory

| Key        | Value Type | Description                                                |
|------------|------------|------------------------------------------------------------|
| midiIn     | string     | name of the midi port that you want to receive input from  |
| oscOutIP   | string     | ip address to send osc messages to                         |
| oscOutPort | int        | port to send orc messages to                               |

## output msc message format:
```
address = /msc/<command>/<cue number>
message = <cue number> true <command>
```

## example packets:
midi input: `F0 7F 01 02 01 01 32 36 35 00 31 00 F7`  (go cue 265 A/B fader)

osc output: `/msc/go/265 ,iTs 265 true go`