package hash_test

import (
	"testing"

	"audio/internal/hash"
)

// TestNormMatchesDispatcher pins the normalization shared with the
// dispatcher synth_audio dedup: " ".join(text.lower().split()).
func TestNormMatchesDispatcher(t *testing.T) {
	if got := hash.Normalize("  Да.\n\tНЕ  знаю… "); got != "да. не знаю…" {
		t.Fatalf("norm = %q", got)
	}
}

// TestTexthashBindsVoice pins voice+rate+model inside the digest:
// a voice change yields new files, never silent replacement.
func TestTexthashBindsVoice(t *testing.T) {
	a := hash.Texthash("да.", "kseniya", 8000, hash.Model)
	b := hash.Texthash("да.", "xenia", 8000, hash.Model)
	if a == b {
		t.Fatal("voice not bound into texthash")
	}
	c := hash.Texthash("да.", "kseniya", 16000, hash.Model)
	if a == c {
		t.Fatal("rate not bound into texthash")
	}
}

func TestS3KeyLayout(t *testing.T) {
	sum := hash.Texthash("да.", "kseniya", 8000, hash.Model)
	hexDigest := hash.Hex(sum)
	key := hash.S3Key("kseniya", 8000, hexDigest)
	want := "v1/kseniya/8000/" + hexDigest[:2] + "/" + hexDigest[2:4] + "/" + hexDigest + ".wav"
	if key != want {
		t.Fatalf("s3key = %q", key)
	}
}
func TestRoundtrip(t *testing.T) {
	sum := hash.Texthash("контейнер горит.", "kseniya", 8000, hash.Model)
	hexed := hash.Hex(sum)
	back, err := hash.Decode(hexed)
	if err != nil || back != sum {
		t.Fatalf("roundtrip: %v", err)
	}
}
