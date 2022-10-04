package main

import (
	"flag"
	"fmt"
)

func main() {
	var file_path string
	flag.StringVar(&file_path, "f", "", "Path to midi file")
	flag.StringVar(&file_path, "file", "", "Path to midi file")

	var bpm uint64
	flag.Uint64Var(&bpm, "b", 120, "Specify custom BPM")
	flag.Uint64Var(&bpm, "bpm", 120, "Specify custom BPM")

	var octave_shift int
	flag.IntVar(&octave_shift, "o", 0, "Shift all notes +/- octaves")
	flag.IntVar(&octave_shift, "octave", 0, "Shift all notes +/- octaves")

	var note_shift int
	flag.IntVar(&note_shift, "n", 0, "Shift all notes +/-")
	flag.IntVar(&note_shift, "note", 0, "Shift all notes +/-")

	var fine_tuning float64
	flag.Float64Var(&fine_tuning, "t", 0, "Shift all notes +/- Hz")
	flag.Float64Var(&fine_tuning, "tuning", 0, "Shift all notes +/- Hz")

	flag.Parse()

	//file_path := "/Users/altucor/projects/personal/programming/go/midi-to-mikrotik-go/input/later_bitches_test.mid"
	midi := MidiFromPath(file_path)
	script := generateScript(midi, 1, 0, bpm, octave_shift, note_shift, fine_tuning, true)
	fmt.Println(script)
}
