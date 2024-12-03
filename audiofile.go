package main

import (
	"os"
	"path/filepath"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	log "github.com/sirupsen/logrus"
)

// play a simple audio file with no fading or level change
func (m *MSCMap) playAudioFile(cue float64) {
	mc, ok := m.midiMap[cue]
	if !ok {
		log.Debugf("did not find cue mapping for cue[%v]", cue)
		return
	}

	filename := mc.audioFile

	if filename == "" {
		log.Debugf("did not find audio file for cue[%v]", cue)
		return
	}

	fileExtension := filepath.Ext(filename)

	if fileExtension != ".mp3" && fileExtension != ".wav" {
		log.Errorf("incompatible file extension: %s", filename)
		return
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Errorf("cannot open file %s: %v", filename, err)
		return
	}

	if fileExtension == ".mp3" {
		streamer, format, err := mp3.Decode(file)
		if err != nil {
			log.Errorf("cannot decode file %s: %v", filename, err)
			return
		}
		defer streamer.Close()

		// buffer size of 1/10 of a second
		resampled := beep.Resample(DefaultResampleQuality, DefaultSampleRate, format.SampleRate, streamer)

		done := make(chan bool)
		speaker.Play(beep.Seq(resampled, beep.Callback(func() {
			done <- true
		})))

		<-done
	}

	if fileExtension == ".wav" {
		streamer, format, err := wav.Decode(file)
		if err != nil {
			log.Errorf("cannot decode file %s: %v", filename, err)
			return
		}
		defer streamer.Close()

		// buffer size of 1/10 of a second
		resampled := beep.Resample(DefaultResampleQuality, DefaultSampleRate, format.SampleRate, streamer)

		done := make(chan bool)
		speaker.Play(beep.Seq(resampled, beep.Callback(func() {
			done <- true
		})))

		<-done
	}
}
