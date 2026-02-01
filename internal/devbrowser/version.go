package devbrowser

import (
	"crypto/sha256"
	"encoding/hex"
)

// DaemonVersion is a compatibility/version stamp used to decide whether an existing
// daemon should be restarted. Keep it stable and purely derived from embedded assets
// + schema-affecting behavior.
func DaemonVersion() string {
	// Include harness init JS content hash so changes force a restart.
	sum := sha256.Sum256([]byte(harnessInitJS))
	return "go-dev-browser-daemon/" + hex.EncodeToString(sum[:8])
}
