package wav_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"audio/internal/wav"
)

func makeWav(n int) []byte {
	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+n))
	buf.WriteString("WAVEfmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(8000))
	_ = binary.Write(buf, binary.LittleEndian, uint32(16000))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(n))
	buf.Write(make([]byte, n))
	return buf.Bytes()
}

func TestCheckOK(t *testing.T) {
	info, err := wav.Check(makeWav(48000))
	if err != nil {
		t.Fatal(err)
	}
	if info.Bytes != 44+48000 || info.DurS != 3.0 {
		t.Fatalf("info = %+v", info)
	}
}

func TestCheckRejects(t *testing.T) {
	if _, err := wav.Check([]byte("short")); err == nil {
		t.Fatal("expected short error")
	}
	bad := makeWav(100)
	bad[24] = 44 // 44100 rate
	if _, err := wav.Check(bad); err == nil {
		t.Fatal("expected rate error")
	}
}
