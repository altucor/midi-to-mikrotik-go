package main

import (
	"fmt"
	"log"
)

const g_unknown_text = "<UNKNOWN>"

func getTimeAsText(time float64) string {
	// HH:MM:SS:MS
	hh := (int(time) / (1000 * 60 * 60)) % 24
	mm := (int(time) / (1000 * 60)) % 60
	ss := (int(time) / 1000) % 60
	ms := int(time) % 1000
	return fmt.Sprintf("%02d:%02d:%02d:%03d", hh, mm, ss, ms)
}

type ScriptHeader struct {
	pulses_per_second float64
	notes_count       uint64
	track_length      float64
	track_name        string
	instrument_name   string
	track_copyright   string
}

func getHeader(cfg BuildConfig, header ScriptHeader) string {
	var outputBuffer = ""
	outputBuffer += "#----------------File Description-----------------#\n"
	outputBuffer += "# This file generated by Midi To Mikrotik Converter\n"
	outputBuffer += "# Visit app repo: https://github.com/altucor/midi-to-mikrotik-go\n"
	outputBuffer += "# Original midi file name/path: " + cfg.file_path + "\n"
	outputBuffer += "# Pulses per second: " + fmt.Sprintf("%f", header.pulses_per_second) + "\n"
	outputBuffer += "# Track index: " + fmt.Sprintf("%d", cfg.track) + "\n"
	outputBuffer += "# MIDI Channel: " + fmt.Sprintf("%d", cfg.channel) + "\n"
	outputBuffer += "# Track BPM: " + fmt.Sprintf("%d", cfg.bpm) + "\n"
	outputBuffer += "# Number of notes: " + fmt.Sprintf("%d", header.notes_count) + "\n"
	outputBuffer += "# Track length: " + getTimeAsText(header.track_length) + " HH:MM:SS:MS\n"
	outputBuffer += "# Track name: " + header.track_name + "\n"
	outputBuffer += "# Instrument name: " + header.instrument_name + "\n"
	outputBuffer += "# Track copyright: " + header.track_copyright + "\n"
	//outputBuffer += "# Track text: " + chunk.mtrkChunkHandler.getTrackText() + "\n";
	//outputBuffer += "# Track copyright: " + chunk.mtrkChunkHandler.getCopyright() + "\n";
	//outputBuffer += "# Vocals: " + chunk.mtrkChunkHandler.getInstrumentName() + "\n";
	//outputBuffer += "# Text marker: " + chunk.mtrkChunkHandler.getTextMarker() + "\n";
	//outputBuffer += "# Cue Point: " + chunk.mtrkChunkHandler.getCuePoint() + "\n";
	outputBuffer += "#-------------------------------------------------#\n\n"
	return outputBuffer
}

func durationToMs(vlv VLV, pulses_per_second float64) float64 {
	return float64(vlv.Value) * pulses_per_second
}

func buildNote(event Event, cfg BuildConfig, pulses_per_second float64, current_time float64) string {
	/*
	 * :beep frequency=440 length=1000ms;
	 * :delay 1000ms;
	 */
	var output string = ""
	freq_cmd, duration := getNote(event)
	freq := g_freqNotes[freq_cmd+(cfg.octave_shift*NOTES_IN_OCTAVE)+cfg.note_shift] + cfg.fine_tuning
	duration_text := fmt.Sprintf("%f", durationToMs(duration, pulses_per_second)) + "ms;"
	if event.Cmd.MainCmd == NOTE_ON {
		output += ":beep frequency=" + fmt.Sprintf("%f", freq) + " length=" + duration_text
		if cfg.comments {
			output += " # " + g_symbolicNotes[freq_cmd+(cfg.octave_shift*NOTES_IN_OCTAVE)+cfg.note_shift]
			if cfg.fine_tuning != 0 {
				output += " " + fmt.Sprintf("%+.3f", cfg.fine_tuning) + "Hz"
			}

			output += " @ " + getTimeAsText(current_time) + "\n"
		} else {
			output += "\n"
		}
	}
	if duration.Value != 0 {
		output += ":delay " + duration_text + "\n"
	}

	return output
}

func generateScript(midi MidiFile, cfg BuildConfig) string {
	var output = ""
	var header ScriptHeader
	header.instrument_name = g_unknown_text
	header.track_name = g_unknown_text
	header.track_copyright = g_unknown_text

	switch midi.Mthd.Format {
	case MIDI_V0:
		// Single track file
		log.Fatal("Single track file")
	case MIDI_V1:
		// First track contains only service info and should be skipped
		if cfg.bpm == 0 {
			tempo_event := midi.Tracks[0].findEventByCmd(MidiEventCodeFromByte(TEMPO))
			cfg.bpm = getEventTempo(tempo_event)
		}
		header.pulses_per_second = (60000.0 / (float64(cfg.bpm) * float64(midi.Mthd.Ppqn)))
		header.instrument_name = getTextFromEvent(midi.Tracks[cfg.track].findEventByCmd(MidiEventCodeFromByte(INSTRUMENT_NAME)))
		header.track_name = getTextFromEvent(midi.Tracks[cfg.track].findEventByCmd(MidiEventCodeFromByte(TRACK_NAME)))
		header.track_copyright = getTextFromEvent(midi.Tracks[cfg.track].findEventByCmd(MidiEventCodeFromByte(COPYRIGHT)))
	case MIDI_V2:
		// At current moment dont know how to behave correctly
		log.Fatal("MIDI_V2")
	default:
		log.Fatal("Unknown midi format detected: ", midi.Mthd.Format)
	}

	var body string = ""
	// TODO: Here add pre-delay
	header.track_length = durationToMs(midi.Tracks[cfg.track].Predelay, header.pulses_per_second)
	var note_on uint64 = 0
	var note_off uint64 = 0
	for i := 0; i < len(midi.Tracks[cfg.track].Events); i++ {
		event := midi.Tracks[cfg.track].Events[i]
		if event.Cmd.MainCmd == NOTE_ON || event.Cmd.MainCmd == NOTE_OFF {
			if event.Cmd.MainCmd == NOTE_ON {
				note_on++
			} else if event.Cmd.MainCmd == NOTE_OFF {
				note_off++
			}
			temp := buildNote(event, cfg, header.pulses_per_second, header.track_length)
			body += temp
		}
		header.track_length += durationToMs(midi.Tracks[cfg.track].Events[i].Delay, header.pulses_per_second)
	}
	if note_on != note_off {
		log.Fatal("Error not equal count of events: note_on = ", note_on, " note_off = ", note_off)
	}

	output += getHeader(cfg, header)
	output += body
	if cfg.print_stdout {
		fmt.Print(output)
	}
	return output
}
