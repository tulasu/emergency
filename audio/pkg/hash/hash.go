// Package hash implements the content-addressed blob key (spec 06):
// norm is the same normalization as the dispatcher synth_audio dedup,
// texthash binds voice+rate+model into the digest so a voice change
// yields new files instead of silently replacing old ones.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Model pins the TTS engine inside the digest.
const Model = "silero-v4_ru"

// Normalize collapses whitespace and lowercases, mirroring
// " ".join(text.lower().split()).
func Normalize(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

// Texthash digests norm|voice|rate|model.
func Texthash(norm, voice string, rate int, model string) [32]byte {
	return sha256.Sum256([]byte(norm + "|" + voice + "|" + fmt.Sprint(rate) + "|" + model))
}

// Hex encodes a digest for manifests, routes and ETags.
func Hex(sum [32]byte) string {
	return hex.EncodeToString(sum[:])
}

// Decode parses a manifest/route hex digest.
func Decode(s string) ([32]byte, error) {
	var sum [32]byte
	b, err := hex.DecodeString(s)
	if err != nil {
		return sum, err
	}
	if len(b) != 32 {
		return sum, fmt.Errorf("bad hash length %d", len(b))
	}
	copy(sum[:], b)
	return sum, nil
}

// S3Key maps a hex digest to v1/{voice}/{rate}/{hh}/{hh}/{hex}.wav.
func S3Key(voice string, rate int, hexDigest string) string {
	return fmt.Sprintf("v1/%s/%d/%s/%s/%s.wav",
		voice, rate, hexDigest[:2], hexDigest[2:4], hexDigest)
}
