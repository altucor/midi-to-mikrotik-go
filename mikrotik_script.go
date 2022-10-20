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
	outputBuffer += "#-------------------------------------------------#\n\n"
	return outputBuffer
}

func durationToMs(vlv VLV, pulses_per_second float64) float64 {
	return float64(vlv.Value) * pulses_per_second
}

func buildNoteDuration(event Event, pulses_per_second float64) string {
	duration := getNoteDuration(event)
	return fmt.Sprintf("%f", durationToMs(duration, pulses_per_second)) + "ms;"
}

func buildDelay(event Event, pulses_per_second float64) string {
	duration := getNoteDuration(event)
	if duration.Value != 0 {
		return ":delay " + buildNoteDuration(event, pulses_per_second) + "\n"
	}
	return ""
}

func buildNote(event Event, cfg BuildConfig, pulses_per_second float64, current_time float64) string {
	/*
	 * :beep frequency=440 length=1000ms;
	 * :delay 1000ms;
	 */
	var output string = ""
	if event.Cmd.MainCmd == NOTE_ON {
		freq_cmd := getNoteFreq(event)
		freq := g_freqNotes[freq_cmd+(cfg.octave_shift*NOTES_IN_OCTAVE)+cfg.note_shift] + cfg.fine_tuning
		output += ":beep frequency=" + fmt.Sprintf("%f", freq) + " length=" + buildNoteDuration(event, pulses_per_second)
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
	output += buildDelay(event, pulses_per_second)
	return output
}

type Note struct {
	start    float64
	end      float64
	freq_cmd int
	velocity int
}

func NoteFromEvent(event Event, time float64) Note {
	note := Note{}
	note.start = time
	note.freq_cmd = getNoteFreq(event)
	note.velocity = getNoteVelocity(event)
	return note
}

func (ctx *Note) isUnfinished(event Event) bool {
	return ctx.freq_cmd == getNoteFreq(event)
}

func (ctx *Note) setEnd(end float64) {
	ctx.end = end
}

type Sequence struct {
	track_name        string
	instrument_name   string
	track_copyright   string
	length            float64
	pulses_per_second float64
	channel           uint
	bpm               uint64
	notes             []Note
}

func (ctx *Sequence) addNote(note Note) {
	ctx.notes = append(ctx.notes, note)
}

func (ctx *Sequence) setEndForUnfinished(event Event, time float64) {
	for index := range ctx.notes {
		if ctx.notes[index].isUnfinished(event) {
			ctx.notes[index].setEnd(time)
		}
	}
}

func generateScript(midi MidiFile, cfg BuildConfig) string {
	var overlay_detect [NOTES_IN_OCTAVE * 10]bool
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

	sequence := Sequence{}
	var unfinished_notes []Note

	var body string = ""
	header.track_length = 0.0
	if midi.Tracks[cfg.track].Predelay.Value != 0 {
		sequence.length += durationToMs(midi.Tracks[cfg.track].Predelay, header.pulses_per_second)
		body += ":delay " + fmt.Sprintf("%f", header.track_length) + "ms;\n"
	}

	var note_on uint64 = 0
	var note_off uint64 = 0

	for i := 0; i < len(midi.Tracks[cfg.track].Events); i++ {
		event := midi.Tracks[cfg.track].Events[i]
		if event.Cmd.MainCmd == NOTE_ON {
			note := NoteFromEvent(event, sequence.length)
			unfinished_notes = append(unfinished_notes, note)
		}
		if event.Cmd.MainCmd == NOTE_OFF {

		}
		if event.Cmd.MainCmd == NOTE_ON || event.Cmd.MainCmd == NOTE_OFF {
			if event.Cmd.SubCmd != uint8(cfg.channel) {
				continue
			}
			note_text := buildNote(event, cfg, header.pulses_per_second, header.track_length)
			if !overlay_detect[getNoteFreq(event)] {
				body += note_text
			} else {
				fmt.Println("Note overlay detected!")
				fmt.Println(note_text)
			}

			if event.Cmd.MainCmd == NOTE_ON {
				overlay_detect[getNoteFreq(event)] = true
				note_on++
			} else if event.Cmd.MainCmd == NOTE_OFF {
				overlay_detect[getNoteFreq(event)] = false
				note_off++
			}
		} else {
			body += buildDelay(event, header.pulses_per_second)
		}
		header.track_length += durationToMs(midi.Tracks[cfg.track].Events[i].Delay, header.pulses_per_second)
	}
	if note_on != note_off {
		log.Fatal("Error not equal count of events: note_on = ", note_on, " note_off = ", note_off)
	}
	header.notes_count = note_on

	output += getHeader(cfg, header)
	output += body
	output += "#---------------------- END ----------------------#\n"
	if cfg.print_stdout {
		fmt.Print(output)
	}

	return output
}
