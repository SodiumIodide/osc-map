package main

import log "github.com/sirupsen/logrus"

// sendKeyboardCommand simulates a keyboard keypress. Useful for soundboard programs
func (m *OSCMap) sendKeyboardCommand(cueNumber string, cueInteger string) {
	if m.keyBonding == nil {
		log.Errorf("keybonding is nil")
		return
	}

	cueMap, ok := m.controlMap[cueNumber]
	if !ok {
		cueMap, ok = m.controlMap[cueInteger]
		if !ok {
			log.Debugf("No virtual keyboard command for cue[%v]", cueNumber)
			return
		}
	}

	if cueMap.keyboardKey == -1 {
		log.Infof("No keyboard key specified for cue[%v]", cueNumber)
		return
	}

	m.keyBonding.SetKeys(cueMap.keyboardKey)

	log.Debugf("Sending keyboard: %v", cueMap.keyboardKey)

	// Press the selected keys
	err := m.keyBonding.Launching()
	if err != nil {
		log.Errorf("failed to launch key: %X", cueMap.keyboardKey)
	}
}
