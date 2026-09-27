// Package wav validates wav blobs before S3 upload: 8kHz mono s16 PCM.
// A corrupt synth output must fail the queue item, not poison the bucket.
package wav

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Info describes a validated wav blob.
type Info struct {
	Bytes int
	DurS  float64
}

// Check parses a minimal RIFF/WAVE header and returns size + duration.
// Layout: RIFF(12) + fmt (24) + data(8) + PCM. synth-daemon writes exactly
// this shape via the wave module (no extra chunks).
func Check(wav []byte) (Info, error) {
	if len(wav) < 44 {
		return Info{}, fmt.Errorf("wav too short: %d", len(wav))
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return Info{}, fmt.Errorf("not a RIFF/WAVE file")
	}
	if string(wav[12:16]) != "fmt " {
		return Info{}, fmt.Errorf("missing fmt chunk")
	}
	if binary.LittleEndian.Uint16(wav[20:22]) != 1 {
		return Info{}, fmt.Errorf("not PCM")
	}
	channels := binary.LittleEndian.Uint16(wav[22:24])
	rate := binary.LittleEndian.Uint32(wav[24:28])
	bits := binary.LittleEndian.Uint16(wav[34:36])
	if channels != 1 || rate != 8000 || bits != 16 {
		return Info{}, fmt.Errorf("want 8kHz mono s16, got %dch %dHz %dbit", channels, rate, bits)
	}
	dataOff := bytes.Index(wav[12:], []byte("data"))
	if dataOff < 0 {
		return Info{}, fmt.Errorf("missing data chunk")
	}
	dataOff += 12
	n := int(binary.LittleEndian.Uint32(wav[dataOff+4 : dataOff+8]))
	if dataOff+8+n > len(wav) {
		return Info{}, fmt.Errorf("truncated data: %d > %d", dataOff+8+n, len(wav))
	}
	if n == 0 {
		return Info{}, fmt.Errorf("empty data")
	}
	return Info{Bytes: len(wav), DurS: float64(n) / 2 / 8000}, nil
}
