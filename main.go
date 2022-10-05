package main

import (
	"flag"
)

type BuildConfig struct {
	file_path    string
	track        uint
	channel      uint
	bpm          uint64
	octave_shift int
	note_shift   int
	fine_tuning  float64
	comments     bool
	print_stdout bool
}

func main() {

	var cfg BuildConfig
	// TODO: Find how to parse correctly short and long types of arguments
	flag.StringVar(&cfg.file_path, "file", "", "Path to midi file")
	flag.UintVar(&cfg.track, "track", 0, "Select track from which extract notes")
	flag.UintVar(&cfg.channel, "channel", 0, "Select channel from which extract notes")
	flag.Uint64Var(&cfg.bpm, "bpm", 0, "Specify custom BPM")
	flag.IntVar(&cfg.octave_shift, "octave", 0, "Shift all notes +/- octaves")
	flag.IntVar(&cfg.note_shift, "note", 0, "Shift all notes +/-")
	flag.Float64Var(&cfg.fine_tuning, "fine", 0, "Shift all notes +/- Hz")
	flag.BoolVar(&cfg.comments, "comments", false, "Enable comments for each note")
	flag.BoolVar(&cfg.print_stdout, "print", false, "Print output to stdout")

	flag.Parse()

	midi := MidiFromPath(cfg.file_path)
	generateScript(midi, cfg)
}
