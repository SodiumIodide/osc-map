package main

import log "github.com/sirupsen/logrus"

// sendKeyboardCommand simulates a keyboard keypress. Useful for soundboard programs
func (m *MSCMap) sendKeyboardCommand(cue float64) {
	if m.keyBonding == nil {
		log.Errorf("keybonding is nil")
		return
	}

	cueMap, ok := m.midiMap[cue]
	if !ok {
		log.Debugf("no virtual keyboard command for cue[%v]", cue)
		return
	}

	if cueMap.keyboardKey == -1 {
		log.Debugf("no keyboard key specified for cue[%v]", cue)
		return
	}

	m.keyBonding.SetKeys(cueMap.keyboardKey)

	log.Debugf("sending keyboard: %v", cueMap.keyboardKey)

	// Press the selected keys
	err := m.keyBonding.Launching()
	if err != nil {
		log.Errorf("failed to launch key: %X", cueMap.keyboardKey)
	}
}
