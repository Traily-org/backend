package ban

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/traily-org/server/internal/domain/user"
)

// NormalizeEmail returns the form of email used to detect a banned address
// under another spelling: trimmed and lowercased, without a "+tag" in the
// local part, and for Gmail without dots and with googlemail.com folded into
// gmail.com. Those variants all reach the same mailbox.
func NormalizeEmail(email string) string {
	e := user.NormalizeEmail(email)

	at := strings.LastIndexByte(e, '@')
	if at <= 0 {
		return e
	}
	local, domain := e[:at], e[at+1:]

	if plus := strings.IndexByte(local, '+'); plus > 0 {
		local = local[:plus]
	}
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	return local + "@" + domain
}

// HashEmail returns the hex SHA-256 of the normalized email. Only this hash is
// stored, so the list of banned identifiers does not hold readable addresses.
func HashEmail(email string) string {
	sum := sha256.Sum256([]byte(NormalizeEmail(email)))
	return hex.EncodeToString(sum[:])
}
