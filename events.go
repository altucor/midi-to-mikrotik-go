package main

func getEventTempo(event Event) uint64 {
	//FF 51 03 tt tt tt
	var tempo uint64
	tempo = uint64(event.Data[0]) << 16
	tempo |= uint64(event.Data[1]) << 8
	tempo |= uint64(event.Data[2])
	tempo = 60000000 / tempo
	return tempo
}

// freq_cmd, VLV
func getNote(event Event) (int, VLV) {
	return int(event.Data[0]), event.Delay
}

func getTextFromEvent(event Event) string {
	if len(event.Data) == 0 {
		return g_unknown_text
	}
	return string(event.Data)
}
