package main

import (
	"flag"
	"fmt"
	"os"
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

	var analyze bool = false
	flag.BoolVar(&analyze, "analyze", false, "Analyze midi track")
	var process_all bool = false
	flag.BoolVar(&process_all, "all", false, "Process all tracks")

	flag.Parse()

	midi := MidiFromPath(cfg.file_path)
	if analyze {
		auto_cfg := cfg
		analysis := midi.analyze()
		for track_index, track := range analysis {
			for channel_index, channel := range track.channels {
				fmt.Println("# Track index:", track_index, "Track name:", track.name, "Found notes ON/OFF:", channel["on"], "/", channel["off"])
				if process_all {
					fmt.Println("# Processing...")
					auto_cfg.channel = uint(channel_index)
					auto_cfg.track = uint(track_index)
					//sequence := generateMikrotikSequence(midi, auto_cfg)
					//script_out := sequence.script(auto_cfg)
					//fmt.Println(script_out)
					script := generateScript(midi, auto_cfg)
					err := os.WriteFile(cfg.file_path+fmt.Sprintf("_%d", auto_cfg.track)+".txt", []byte(script), 0644)
					check(err)
				}
			}
		}
	} else {
		output_script := generateScript(midi, cfg)
		err := os.WriteFile(cfg.file_path+fmt.Sprintf("_%d", cfg.track)+".txt", []byte(output_script), 0644)
		check(err)
	}
}
