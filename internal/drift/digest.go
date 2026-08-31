package drift

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "sha256:invalid"
	}
	return DigestBytes(data)
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || value[:len("sha256:")] != "sha256:" {
		return false
	}
	_, err := hex.DecodeString(value[len("sha256:"):])
	return err == nil
}

func envelopeDigest(envelope Envelope) string {
	copy := envelope
	copy.ReleaseDigest = ""
	return DigestJSON(copy)
}
