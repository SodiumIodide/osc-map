# msc-map

Receives MSC (MIDI Show Control) messages from an etc express (sysex messages), sends it out as an osc message, and sends out a midi program change message. Tested on Windows. This project has been forked to support the Los Alamos Little Theater in receiving MSC signal from the lightboard and sending a MIDI signal to the soundboard for cue sync and mapping.

## Dependencies

- requires CGO

## Build

Execute `go build` in the local working directory.

## Config

msc-map will look for `config.yaml` in the local directory.

## File construction

For LALT use with current equipment (e.g. tt24 soundboard and etc express lightboard), the config file should start with this header:

```yaml
midiIn: "USB MIDI Interface"

outputs:
    osc:
        ip: 127.0.0.1
        port: 8765
    midi-pc:
        name: UM-ONE
        channel: 1
    qlab: false
    keyboard-commands: true
    audio-files: true
```

If no key-mapping or audio playback is required for the current project, the values of `keyboard-commands` and `audio-files` respectively may be set to `false`, but any performance impact of allowing them is inconsequential.

Following the above header, a new YAML list may be constructed titled `midi-cue-mapping`. This is where the bulk of the project will be constructed. Each entry in this list should start with a cue number corresponding to the cue on the lightboard input as a `light` value with an integer. For each light cue, there are several options to attach to that signal:

- `sound`
- `unmute`
- `mute`
- `fader`
  - `value`
- `keyboard`
- `file`

### `sound` - Integer

The `sound` option corresponds to a snapshot number on the soundboard. When the corresponding light cue is received, the soundboard will load the snapshot number specified by the number provided. There is some latency to this command which is endemic to the soundboard firmware itself.

### `unmute` - \[Integer\]

The `unmute` option will unmute the channel IDs matching the numbers provided.

### `mute` - \[Integer\]

The `mute` option will mute the channel IDs matching the numbers provided.

### `fader` - \[Integer\] & `value` - \[Integer\]

The `fader` option will receive an array of integer numbers corresponding to channel IDs. Proper use of this option also requires an equal number of `value` integers to be provided. The `value` is the relative volume level that the fader will be set to. The `value` can be in the range of 0 to 127, with 100 corresponding to the 0 or "neutral" dB volume level on the fader, represented by the signal "U" on the tt24 faders. Similarly, using a `value` of 0 will "minimize" the fader, setting the volume to negative infinity, effectively muting the channel. Using a `value` of 127 will "maximize" the fader, setting the volume to +10 dB. As a note, the fader levels are in logarithmically scaled steps.

### `keyboard` - String

The `keyboard` option will deliver a keypress to the computer's operating system, acting as if a physical key on the keyboard was pressed. The utility of this is in utilizing various audio cue programs such as SCS where cue triggers can be tied to keypresses. For example, you may use SCS to set an intermission track to fade in after pressing the key "J", and then when intermission is over you may set a subcue to fade it out that triggers when pressing "K". Using the `keyboard` option allows for this behavior to be automated via signals sent from lightboard cues.

Allowable keys are a-z, A-Z, and 0-9, entered in quotation marks in the value of this field. Additional "control" keys for cue software that might support them are as follows:

- Backspace : `"bs"` or `"BS"`
- Enter : `"ent"` or `"ENT"`
- Escape: `"esc"` or `"ESC"`

### `file` - String

The `file` option will trigger simple playback of an audio file given a path location. As there is no control for fine volume elements like adding a fade, stop, or volume level select; this option is best used for "simple" sound effects that do not require any control, e.g. doorbells, gunshots, etc. Use this option for sounds that are "fire and forget", and do not need further input other than starting and letting the track finish. This player supports `mp3` and `wav` files. Other audio formats should be played in external audio cue software.

The option given here is a file path as a string, meaning in quotation marks. You can easily obtain this in Windows by navigating to the file you wish to play, right clicking it, and selecting the "copy as path" context option. It's important to note that the YAML syntax used requires Windows path delineators (backslashes) to be "escaped" by adding another backslash, so that they are correctly interpreted as backslashes and not other YAML characters. An example path that is correctly escaped would look like:

`"C:\\Users\\LALT\\Documents\\Shows\\ThePlayThatGoesWrong_SFX\\door-chime.mp3"`

## Example

As an example, consider a simple cue program.

- On light cue 1, we want to unmute sound channel 4, set the volume level of sound channel 1 to 0 dB, and play an audio file.
- On light cue 2, we want to mute sound channel 4, and set the volume level of sound channel 2 to 50.
- On light cue 5, we want to play an audio file.
- On light cue 13, we want to change the soundboard to snapshot 5, and trigger keypress "J" to play a file in our audio cue software.
- On light cue 20, we want to trigger keypress "ESC" to stop the track playing in our audio cue software.

The corresponding `config.yaml` file might look like the following, with comments to notate the show flow preceded by a `#` symbol:

```yaml
midiIn: "USB MIDI Interface"

outputs:
    osc:
        ip: 127.0.0.1
        port: 8765
    midi-pc:
        name: UM-ONE
        channel: 1
    qlab: false
    keyboard-commands: true
    audio-files: true

midi-cue-mapping:
    # Show opening, raise volume for dialogue on channel 1 and unmute 4 for
    # entrance while doorbell rings.
    - light: 1
      unmute: [4]
      fader: [1]
      value: [100]
      file: "C:\\Users\\LALT\\Documents\\Shows\\MyCoolShow\\door-chime.mp3"
    # Next scene, mute 1 and 4, and bring channels 2 and 3 down to half volume because the
    # actors yell for this part, bring channel 5 to neutral volume
    - light: 2
      mute: [1, 4]
      fader: [2, 3, 5]
      value: [50, 50, 100]
    # Murder happens!
    - light: 5
      file: "C:\\Users\\LALT\\Documents\\Shows\\MyCoolShow\\gunshot.wav"
    # For the intermission, use new soundboard snapshot and play fun music
    # in audio cue software.
    - light: 13
      sound: 5
      keyboard: "J"
    # Stop the music.
    - light: 20
      keyboard: "ESC"
```

Comments on the cues are not necessary but can help to delineate the file and understand the flow of the cue mapping.

## Brief variable description

| Key                       | Value Type | Description                                                                                                |
|---------------------------|------------|------------------------------------------------------------------------------------------------------------|
| midiIn                    | string       | name of the midi port that you want to receive input from                                                  |
| outputs.osc.ip            | ip address   | the ip address to send osc messages to                                                                     |
| outputs.osc.port          | int          | the port of to send osc messages to                                                                        |
| outputs.midi-pc.name      | string       | name of the midi port that you want to send program change messages to                                     |
| outputs.midi-pc.channel   | int          | the midi channel that you want to send program change messages to                                          |
| outputs.qlab              | boolean      | true or false depending on if you want to send program change messages to qlab running on the same machine |
| midi-cue-mapping          | array        | list of midi cue mappings                                                                                  |
| midi-cue-mapping.light    | double       | the light cue to listen for from the etc express light board                                               |
| midi-cue-mapping.sound    | int          | the program change cue to send to the tt24 sound board to change soundboard snapshot                       |
| midi-cue-mapping.unmute   | Array\[int\] | the tt24 channel to unmute                                                                                 |
| midi-cue-mapping.mute     | Array\[int\] | the tt24 channel to mute                                                                                   |
| midi-cue-mapping.fader    | Array\[int\] | the tt24 channel to adjust the fader value of                                                              |
| midi-cue-mapping.value    | Array\[int\] | if adjusting a fader, the value to set it at from 0-127                                                    |
| midi-cue-mapping.keyboard | string       | a keypress to trigger on the local machine                                                                 |
| midi-cue-mapping.file     | string       | path to an mp3 or wav file to play                                                                         |

## Output msc message format

`address = /msc/<command>/<cue number>`
`message = <cue number> true <command>`

## Example packets:

midi input: `F0 7F 01 02 01 01 32 36 35 00 31 00 F7`  (go cue 265 A/B fader)

osc output: `/msc/go/265 ,iTs 265 true go`
