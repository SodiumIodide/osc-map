package main

import (
	log "github.com/sirupsen/logrus"
	"gitlab.com/gomidi/midi/v2"
)

// sendMidiOut sends a MIDI message to the midi out that configured in the config
func (m *OSCMap) sendMidiOut(cueNumber string, cueInteger string) {
	if m.midiOut == nil {
		return
	}

	mc, ok := m.controlMap[cueNumber]
	if !ok {
		mc, ok = m.controlMap[cueInteger]
		if !ok {
			log.Debugf("No soundboard interface command for cue[%v]", cueNumber)
			return
		}
	}

	soundCue := mc.soundCue
	muteCue := mc.muteCue
	unmuteCue := mc.unmuteCue
	faderCue := mc.faderCue
	faderVal := mc.faderVal
	scsCue := mc.scs

	if soundCue == 0 && len(muteCue) == 0 && len(unmuteCue) == 0 && len(faderCue) == 0 {
		return
	}

	if soundCue != 0 {
		mm := midi.ProgramChange(m.midiOutChannel, soundCue-1)
		out, err := midi.SendTo(*m.midiOut)
		if err != nil {
			log.Errorf("Failed to get midi send function: %v", err)
			return
		}

		err = out(mm)
		if err != nil {
			log.Errorf("Failed to send midi program change message to [%v]: %v", m.midiOut, err)
			return
		}

		log.Infof("Sent program change %v to midi out", soundCue)
	}

	if len(muteCue) != 0 {
		for i := 0; i < len(muteCue); i++ {
			mm := midi.NoteOn(m.midiOutChannel, muteCue[i]-1, 0x7F)

			out, err := midi.SendTo(*m.midiOut)
			if err != nil {
				log.Errorf("Failed to get midi send function: %v", err)
			}

			err = out(mm)
			if err != nil {
				log.Errorf("Failed to send midi note message to [%v]: %v", m.midiOut, err)
				return
			}
		}

		log.Infof("Sent mute note %v to midi out", muteCue)
	}

	if len(unmuteCue) != 0 {
		for i := 0; i < len(unmuteCue); i++ {
			mm := midi.NoteOn(m.midiOutChannel, unmuteCue[i]-1, 0x00)

			out, err := midi.SendTo(*m.midiOut)
			if err != nil {
				log.Errorf("Failed to get midi send function: %v", err)
			}

			err = out(mm)
			if err != nil {
				log.Errorf("Failed to send midi note message to [%v]: %v", m.midiOut, err)
				return
			}
		}

		log.Infof("Sent unmute note %v to midi out", unmuteCue)
	}

	// Fader value can vary from 0 to 127, where 100 = U
	if len(faderCue) != 0 {
		if len(faderCue) != len(faderVal) {
			log.Errorf("Each fader cue needs a fader value on cue[%v]", cueNumber)
		}
		for i := 0; i < len(faderCue); i++ {
			if faderVal[i] > 127 {
				log.Errorf("Fader value cannot be higher than 127 on cue[%v]", cueNumber)
			}

			mm := midi.ControlChange(m.midiOutChannel, faderCue[i]-1, faderVal[i])

			out, err := midi.SendTo(*m.midiOut)
			if err != nil {
				log.Errorf("Failed to get midi send function: %v", err)
			}

			err = out(mm)
			if err != nil {
				log.Errorf("Failed to send midi control change to [%v]: %v", m.midiOut, err)
				return
			}
		}

		log.Infof("Sent fader value %v, %v control change to midi out", faderCue, faderVal)
	}

	if m.qlabOut != nil {
		mm := midi.ProgramChange(m.midiOutChannel, soundCue)

		out, err := midi.SendTo(*m.qlabOut)
		if err != nil {
			log.Errorf("Failed to get midi send function: %v", err)
		}

		err = out(mm)
		if err != nil {
			log.Errorf("Failed to send midi program change message to [%v]: %v", m.qlabOut, err)
			return
		}

		log.Infof("Sent program change %v to qlab", soundCue)
	}

	if scsCue != 0 {
		mm := midi.NoteOn(m.scsOutChannel, scsCue, 0x01)

		out, err := midi.SendTo(*m.scsOut)
		if err != nil {
			log.Errorf("Failed to get midi send function: %v", err)
		}

		err = out(mm)
		if err != nil {
			log.Errorf("Failed to send midi NoteOn message to [%v]: %v", m.scsOut, err)
			return
		}

		log.Infof("Sent NoteOn signal %v to SCS output [%v]", scsCue, m.scsOut)
	}
}
