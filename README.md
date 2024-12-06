# OSC-Map

Receives OSC (Open Show Control) messages from an ETC ColorSource AV lightboard, and sends out a midi program change message to be interpreted by a Mackie TT24 soundboard. Tested on Windows. This project has been forked and heavily modified to support the Los Alamos Little Theater in receiving OSC signals from the lightboard and sending a MIDI signal to the soundboard for cue sync and mapping.

## Dependencies

- requires CGO

## Build

Execute `go build` in the local working directory.

## Config

OSC-Map will look for `config.yaml` in the local directory.

## File construction

For LALT use with current equipment (e.g. TT24 soundboard and Colorsource AV lightboard), the config file should start with this header:

```yaml
oscIn:
  ip: 10.1.10.203
  port: 8006

outputs:
  oscOut:
    ip: 10.1.10.77
    port: 8005
  midi-pc:
    name: UM-ONE
    channel: 1
  qlab: false 
  keyboard-commands: true
  audio-files: true
```

If no key-mapping or audio playback is required for the current project, the values of `keyboard-commands` and `audio-files` respectively may be set to `false`, but any performance impact of allowing them is inconsequential.

To open a loopback relay with the ColorSource AV lightboard, a ping message must be sent from the program. If no ping response is received, check the lightboard's connectivity to the network, as well as the gateway IP and port and ensure that the operating computer and the lightboard are on the same subnet mask (typically /24, or 255.255.255.0).

Following the above header, a new YAML list may be constructed titled `control-cue-mapping`. This is where the bulk of the project will be constructed. Each entry in this list should start with a cue number corresponding to the cue on the lightboard input as a `light` value with a numerical string. Support light cue numbers include integers (e.g. 1, 5, 14), decimal integers (e.g. 1.0, 5.0, 14.0), and single-digit decimals (e.g. 1.1, 5.6, 14.9). For each light cue, there are several options to attach to that signal:

- `sound`
- `unmute`
- `mute`
- `fader`
  - `value`
- `keyboard`
- `file`
- `houselights`
  - `rgbws`
  - `transitions`
  - `effects`

***NOTE***
It is very important to keep a consistent spacing and hypen delineation in this file. If a `failure to unmarshal config file` error is shown when executing from a command prompt like PowerShell, users should double-check the indentation of all entries in the file.

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
- Space: `"space"` or `"SPACE"`
- Up Arrow Key: `"up"` or `"UP"`
- Down Arrow Key: `"down"` or `"DOWN"`
- Left Arrow Key: `"left"` or `"LEFT"`
- Right Arrow Key: `"right"` or `"RIGHT"`

### `file` - String

The `file` option will trigger simple playback of an audio file given a path location. As there is no control for fine volume elements like adding a fade, stop, or volume level select; this option is best used for "simple" sound effects that do not require any control, e.g. doorbells, gunshots, etc. Use this option for sounds that are "fire and forget", and do not need further input other than starting and letting the track finish. This player supports `mp3` and `wav` files. Other audio formats should be played in external audio cue software.

The option given here is a file path as a string, meaning in quotation marks. You can easily obtain this in Windows by navigating to the file you wish to play, right clicking it, and selecting the "copy as path" context option. It's important to note that the YAML syntax used requires Windows path delineators (backslashes) to be "escaped" by adding another backslash, so that they are correctly interpreted as backslashes and not other YAML characters. An example path that is correctly escaped would look like:

`"C:\\Users\\LALT\\Documents\\Shows\\ThePlayThatGoesWrong_SFX\\door-chime.mp3"`

### `houselights` - \[Integer\] & `rgbws` \[\[Integer\],...\] & `transitions` \[Float\] & `effects` \[String\]

The `houselights` option will take a list of integers corresponding to house light numbers. The house lights are numbered according to the following schema:

| Stage | Stage | Stage | Stage |
| ----- | ----- | ----- | ----- |
| 1     | 2     | 3     | 4     |
| Joist | Joist | Joist | Joist |
| 5     | 6     | 7     | 8     |
| 9     | 10    | 11    | 12    |
| Booth | 13    | 14    | 15    |

Any numbers greater than 20 will not be functional, and any numbers greater than 15 will likely not be installed in the house unless hardware changes are made. As an aside, any hardware changes should be flashed with similar firmware in HomeAssistant to the existing bulb devices.

Along with the `houselights` list, you can include a 4 digit RGBW value in the form of an integer list from 0-255 for each value to assign a color profile for the specified LED bulbs via the `rgbws` option. As such, this list can range from `[0, 0, 0, 0]` for no light effect to `[255, 255, 255, 255]` for a full light effect. There are theories and sciences behind mixing RGBW values which are outside the scope of this README, so experimentation is encouraged.

The `transitions` list requires integer inputs that correspond to the number of seconds that it takes for the light transition to occur. Note that LED light bulbs can have unexpected color variations due to differences in firmware programming when applying transition length effects. If precise color control is extremely important, it may be best to stick to transition times of 0. If smoothness of lighting effects is desired, then experimentation may be required with RGBW values and transition times to limit any unwanted color variations from the scene.

The final list useful for unitary control of the house lights is the `effects` list, which is a list of strings. Generally, the strings used should be `"None"` or `"Light Board Control"`. To allow for unitary control of the house lights, the effect `"None"` must be used such that the DMX signals emitted from the lightboard do not override the selected `rgbws` values. To relinquish unitary control via this program and allow for lightboard signals to effect the full universe of house lights again, please pass in the `"Light Board Control"` effect with a cue.

Other effects are possible but likely of limited utility during shows, such as `"Strobe"`, `"Fast Rainbow"`, `"Slow Rainbow"`, and others. These may be viewed in the ESPHome YAML files within the HomeAssistant server configuration. Experienced developers may also find a `customRainbow()` function in the `main.go` file, which will require re-compilation to alter.

If all lights in the list given by the `houselights` definition receive the same effect, it is possible to omit repeating the rgbw, effect, and transition lists and just include one value. e.g.:

```yaml
- light: 25
  houselights: [10, 11, 12]
  rgbws: [[255, 70, 0, 255]]
  transitions: [0]
  effects: ["None"]
```

The above snippet will remove lightboard control and allocate the designated RGBW values to house lights 10, 11, and 12 with no transition time on lightboard cue 25. Please remember to re-enable lightboard DMX control at a later cue if necessary, by doing something similar:

```yaml
- light: 26
  houselights: [10, 11, 12]
  rgbws: [[0, 0, 0, 0]]
  transitions: [3]
  effects: ["Light Board Control"]
```

If multiple controls and transition times are desired for multiple different lights, they will correspond to the positions in the list provided by the `houselight` variable. Mixing and matching transitions and effects is possible.

A quick snippet for using multiple lights also follows for edification, wherein we will bring the front of house lights to full so as to demonstrate the use of longer lists:

```yaml
- light: 1
  houselights: [1, 2, 3, 4]
  rgbws: [[255, 255, 255, 255], [255, 255, 255, 255], [255, 255, 255, 255], [255, 255, 255, 255]]
  transitions: [3, 2, 2, 3]
  effects: ["None", "None", "None", "None"]
```

## Example

As an example, consider a simple cue program.

- On light cue 1, we want to unmute sound channel 4, set the volume level of sound channel 1 to 0 dB, and play an audio file.
- On light cue 2, we want to mute sound channel 4, and set the volume level of sound channel 2 to 50.
- On light cue 5, we want to play an audio file.
- On light cue 13, we want to change the soundboard to snapshot 5, and trigger keypress "J" to play a file in our audio cue software.
- On light cue 20, we want to trigger keypress "ESC" to stop the track playing in our audio cue software.

The corresponding `config.yaml` file might look like the following, with comments to notate the show flow preceded by a `#` symbol:

```yaml
oscIn:
  ip: 10.1.10.203
  port: 8006

outputs:
  oscOut:
    ip: 10.1.10.77
    port: 8005
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
    - light: 2.1
      mute: [1, 4]
      fader: [2, 3, 5]
      value: [50, 50, 100]
    # Murder happens!
    - light: 5
      file: "C:\\Users\\LALT\\Documents\\Shows\\MyCoolShow\\gunshot.wav"
    # For the intermission, use new soundboard snapshot and play fun music
    # in audio cue software.
    - light: 13.4
      sound: 5
      keyboard: "J"
    # Stop the music.
    - light: 20
      keyboard: "ESC"
```

Comments on the cues are not necessary but can help to delineate the file and understand the flow of the cue mapping.

## Brief variable description

| Key                          | Value Type      | Description                                                                                                |
|------------------------------|-----------------|------------------------------------------------------------------------------------------------------------|
| midiIn                       | string          | name of the midi port that you want to receive input from                                                  |
| outputs.osc.ip               | ip address      | the ip address to send osc messages to                                                                     |
| outputs.osc.port             | int             | the port of to send osc messages to                                                                        |
| outputs.midi-pc.name         | string          | name of the midi port that you want to send program change messages to                                     |
| outputs.midi-pc.channel      | int             | the midi channel that you want to send program change messages to                                          |
| outputs.qlab                 | boolean         | true or false depending on if you want to send program change messages to qlab running on the same machine |
| midi-cue-mapping             | array           | list of midi cue mappings                                                                                  |
| midi-cue-mapping.light       | double          | the light cue to listen for from the etc express light board                                               |
| midi-cue-mapping.sound       | int             | the program change cue to send to the tt24 sound board to change soundboard snapshot                       |
| midi-cue-mapping.unmute      | Array\[int\]    | the tt24 channel to unmute                                                                                 |
| midi-cue-mapping.mute        | Array\[int\]    | the tt24 channel to mute                                                                                   |
| midi-cue-mapping.fader       | Array\[int\]    | the tt24 channel to adjust the fader value of                                                              |
| midi-cue-mapping.value       | Array\[int\]    | if adjusting a fader, the value to set it at from 0-127                                                    |
| midi-cue-mapping.keyboard    | string          | a keypress to trigger on the local machine                                                                 |
| midi-cue-mapping.file        | string          | path to an mp3 or wav file to play                                                                         |
| midi-cue-mapping.houselights | Array\[int\]    | list of house light numbers to affect                                                                      |
| midi-cue-mapping.rgbws       | Array\[Array\[int\]\] | list of 4 integers from 0-255 corresponding to an RGBW value to assign to house lights                     |
| midi-cue-mapping.transitions | Array\[float\]    | transition length in seconds for LED house light bulbs to new RGBW values                                  |
| midi-cue-mapping.effects     | Array\[string\] | effect present on the house lights provided, usually either "None" or "Light Board Control"                |

## Output msc message format

`address = /msc/<command>/<cue number>`
`message = <cue number> true <command>`

## Example packets

midi input: `F0 7F 01 02 01 01 32 36 35 00 31 00 F7`  (go cue 265 A/B fader)

osc output: `/msc/go/265 ,iTs 265 true go`
