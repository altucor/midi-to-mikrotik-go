package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
)

const g_mthd_header string = "MThd"
const g_mtrk_header string = "MTrk"

const (
	MIDI_V0 = 0 // Single track SMF
	MIDI_V1 = 1 // Multi-Track SMF, with first track reserved for service commands
	MIDI_V2 = 2
)

// High part of 4 bits
const (
	UNKNOWN                        = 0x00
	UNK_1                          = 0x01
	UNK_2                          = 0x02
	UNK_3                          = 0x03
	UNK_4                          = 0x04
	UNK_5                          = 0x05
	UNK_6                          = 0x06
	UNK_7                          = 0x07
	NOTE_OFF                       = 0x08 // low 4 bits - channel
	NOTE_ON                        = 0x09 // low 4 bits - channel
	POLYPHONIC_AFTERTOUCH_PRESSURE = 0x0A // low 4 bits - channel
	CONTROL_MODE_CHANGE            = 0x0B // low 4 bits - channel
	PROGRAM_CHANGE                 = 0x0C // low 4 bits - channel
	CHANNEL_AFTERTOUCH_PRESSURE    = 0x0D // low 4 bits - channel
	PITCH_WHEEL                    = 0x0E // low 4 bits - channel
	SYSTEM_EVENT                   = 0x0F // has sub-commands
)

// Low 4 bits of SYSTEM_EVENT command
const (
	SYS_EXCLUSIVE                    = 0x00
	SYS_MIDI_TIME_CODE_QUARTER_FRAME = 0x01
	SYS_SONG_POSITION_POINTER        = 0x02
	SYS_SONG_SELECT                  = 0x03
	SYS_RESERVED_4                   = 0x04
	SYS_RESERVED_5                   = 0x05
	SYS_TUNE_REQUEST                 = 0x06
	SYS_END_OF_SYSEX                 = 0x07 // EOX
	SYS_REAL_TIME_TIMING_CLOCK       = 0x08
	SYS_REAL_TIME_UNDEFINED_9        = 0x09
	SYS_REAL_TIME_START              = 0x0A
	SYS_REAL_TIME_CONTINUE           = 0x0B
	SYS_REAL_TIME_STOP               = 0x0C
	SYS_REAL_TIME_UNDEFINED_D        = 0x0D
	SYS_REAL_TIME_ACTIVE_SENSING     = 0x0E
	SYS_META_EVENT                   = 0x0F
)

const (
	SEQUENCE_NUMBER    = 0x00
	TEXT               = 0x01
	COPYRIGHT          = 0x02
	TRACK_NAME         = 0x03
	INSTRUMENT_NAME    = 0x04
	LYRIC_TEXT         = 0x05 // A single Lyric MetaEvent should contain only one syllable
	TEXT_MARKER        = 0x06
	CUE_POINT          = 0x07
	PROGRAM_PATCH_NAME = 0x08
	DEVICE_PORT_NAME   = 0x09
	MIDI_CHANNEL       = 0x20
	MIDI_PORT          = 0x21
	TRACK_END          = 0x2F
	TEMPO              = 0x51
	SMPTE_OFFSET       = 0x54
	TIME_SIGNATURE     = 0x58
	KEY_SIGNATURE      = 0x59
	PROPRIETARY_EVENT  = 0x7F
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func readFromStream(stream *bytes.Reader, size int) []byte {
	data := make([]byte, size)
	read, err := stream.Read(data)
	check(err)
	if read != size {
		log.Fatal("Stream Error sizes incorrect")
	}
	return data
}

type MthdHeader struct {
	Header     uint32
	Length     uint32
	Format     uint16
	Mtrk_count uint16
	Ppqn       uint16
}

type MidiEventCode struct {
	MainCmd uint8
	SubCmd  uint8
}

func (ctx *MidiEventCode) isMetaEvent() bool {
	return ctx.MainCmd == SYSTEM_EVENT && ctx.SubCmd == SYS_META_EVENT
}

func (ctx *MidiEventCode) fullCmd() uint8 {
	return (ctx.MainCmd << 4) | ctx.SubCmd
}

func MidiEventCodeFromByte(cmd_byte uint8) MidiEventCode {
	midi_event_code := MidiEventCode{}
	midi_event_code.MainCmd = ((cmd_byte >> 4) & 0x0F)
	midi_event_code.SubCmd = (cmd_byte & 0x0F)
	return midi_event_code
}

func MidiEventCodeFromStream(track_stream *bytes.Reader) MidiEventCode {
	midi_event_code := MidiEventCode{}
	cmd_byte := readFromStream(track_stream, 1)[0]

	midi_event_code.MainCmd = ((cmd_byte >> 4) & 0x0F)
	midi_event_code.SubCmd = (cmd_byte & 0x0F)
	return midi_event_code
}

const MIDI_VLV_CONTINUATION_BIT uint8 = 0x80
const MIDI_VLV_DATA_MASK uint8 = 0x7F

// VLV - Variable Length Value
type VLV struct {
	Value uint32
}

func VLVFromStream(track_stream *bytes.Reader) VLV {
	vlv := VLV{}
	for {
		vlv.Value = (vlv.Value << 7)
		vlv_byte := readFromStream(track_stream, 1)[0]

		vlv.Value += uint32(vlv_byte & MIDI_VLV_DATA_MASK)
		if vlv_byte&MIDI_VLV_CONTINUATION_BIT == 0 {
			break
		}
	}

	return vlv
}

// TODO: Rewrite all stuff to process delays as prefix of event, check kaitai implementation
type Event struct {
	MetaEvent bool
	Cmd       MidiEventCode
	Data      []byte
	Delay     VLV
}

func EventFromStream(track_stream *bytes.Reader) Event {
	event := Event{}
	event.Cmd = MidiEventCodeFromStream(track_stream)
	//fmt.Println(event.Cmd.fullCmd())

	if event.Cmd.isMetaEvent() {
		event.MetaEvent = true
		event.Cmd = MidiEventCodeFromStream(track_stream)
		// read vlv meta event size
		data_size := VLVFromStream(track_stream)
		if event.Cmd.fullCmd() == TRACK_END && data_size.Value == 0 {
			return event
		}
		event.Data = readFromStream(track_stream, int(data_size.Value))
	} else {
		event.Data = readFromStream(track_stream, 1)
		switch event.Cmd.MainCmd {
		case NOTE_OFF:
			event.Data = append(event.Data, readFromStream(track_stream, 1)[0])
		case NOTE_ON:
			event.Data = append(event.Data, readFromStream(track_stream, 1)[0])
		case POLYPHONIC_AFTERTOUCH_PRESSURE:
			event.Data = append(event.Data, readFromStream(track_stream, 1)[0])
		case CONTROL_MODE_CHANGE:
			event.Data = append(event.Data, readFromStream(track_stream, 1)[0])
		case PITCH_WHEEL:
			event.Data = append(event.Data, readFromStream(track_stream, 1)[0])
		default:
			break
		}
	}
	event.Delay = VLVFromStream(track_stream)
	return event
}

type MtrkHeader struct {
	Header uint32
	Length uint32
}

type MidiTrack struct {
	MtrkHeader MtrkHeader
	Predelay   VLV
	Events     []Event
}

func MidiTrackFromStream(file *os.File) MidiTrack {
	track := MidiTrack{}
	err := binary.Read(file, binary.BigEndian, &track.MtrkHeader)
	check(err)
	temp := make([]byte, 4)
	binary.BigEndian.PutUint32(temp, track.MtrkHeader.Header)
	if string(temp) != g_mtrk_header {
		log.Fatal("Error invalid MTrk header")
	}
	track_payload := make([]byte, track.MtrkHeader.Length)
	read, err := file.Read(track_payload)
	check(err)
	if read != int(track.MtrkHeader.Length) {
		log.Fatal("Error sizes incorrect")
	}
	// read VLV pre-delay
	track_stream := bytes.NewReader(track_payload)
	track.Predelay = VLVFromStream(track_stream)

	for track_stream.Len() > 0 {
		// read events until end of available
		track.Events = append(track.Events, EventFromStream(track_stream))
	}
	return track
}

func (ctx *MidiTrack) findEventByCmd(cmd MidiEventCode) Event {
	for i := 0; i < len(ctx.Events); i++ {
		if ctx.Events[i].Cmd == cmd {
			return ctx.Events[i]
		}
	}
	return Event{}
}

type TrackAnalysis struct {
	name     string
	channels map[int]map[string]int
}

func (ctx *MidiTrack) analyze() TrackAnalysis {
	analysis := TrackAnalysis{}
	analysis.name = getTextFromEvent(ctx.findEventByCmd(MidiEventCodeFromByte(TRACK_NAME)))
	analysis.channels = make(map[int]map[string]int)
	for i := 0; i < len(ctx.Events); i++ {
		event := ctx.Events[i]
		// for each event
		if event.Cmd.MainCmd == NOTE_ON || event.Cmd.MainCmd == NOTE_OFF {
			if _, ok := analysis.channels[int(event.Cmd.SubCmd)]; !ok {
				analysis.channels[int(event.Cmd.SubCmd)] = make(map[string]int)
			}
			if event.Cmd.MainCmd == NOTE_ON {
				analysis.channels[int(event.Cmd.SubCmd)]["on"]++
			} else if event.Cmd.MainCmd == NOTE_OFF {
				analysis.channels[int(event.Cmd.SubCmd)]["off"]++
			}
		}
	}

	return analysis
}

type MidiFile struct {
	Mthd            MthdHeader
	PulsesPerSedond float64
	Tracks          []MidiTrack
}

func MidiFromPath(path string) MidiFile {
	file, err := os.Open(path)
	check(err)
	defer file.Close()

	midi := MidiFile{}
	err = binary.Read(file, binary.BigEndian, &midi.Mthd)
	check(err)

	temp := make([]byte, 4)
	binary.BigEndian.PutUint32(temp, midi.Mthd.Header)
	if string(temp) != g_mthd_header {
		log.Fatal("Error invalid MThd header")
	}

	for i := 0; i < int(midi.Mthd.Mtrk_count); i++ {
		midi.Tracks = append(midi.Tracks, MidiTrackFromStream(file))
	}
	return midi
}

func (ctx *MidiFile) analyze() []TrackAnalysis {
	var midi_analysis []TrackAnalysis
	for track := 0; track < int(ctx.Mthd.Mtrk_count); track++ {
		// for each track
		midi_analysis = append(midi_analysis, ctx.Tracks[track].analyze())
	}
	return midi_analysis
}
