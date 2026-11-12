package provider

import (
	"crypto/rand"
	"encoding/hex"
)

const openCodeSessionAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// newOpenCodeSessionID builds the native `ses_<12 hex><14 base62>` shape used
// by opencode.ai chat routing. The gateway is stateless, so each upstream
// request gets a fresh session id.
func newOpenCodeSessionID() string {
	random := make([]byte, 20)
	if _, err := rand.Read(random); err != nil {
		return "ses_000000000000000000000000000"
	}
	encoded := make([]byte, 0, 4+12+14)
	encoded = append(encoded, "ses_"...)
	encoded = append(encoded, hex.EncodeToString(random[:6])...)
	for _, value := range random[6:] {
		encoded = append(encoded, openCodeSessionAlphabet[int(value)%len(openCodeSessionAlphabet)])
	}
	return string(encoded)
}
