package peermail

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// auditMonths keeps one parsed month per file, keyed by its path. The audit is
// append-only, so a reader that sees the same size and modification time sees
// the same records; the board and every pane share this instead of re-parsing
// the whole month on every poll. Audit deletes the entry after an append so a
// coarse filesystem timestamp cannot hide a line.
var auditMonths sync.Map

type auditMonth struct {
	mod     time.Time
	size    int64
	records []AuditRecord
}

// Decision is what became of a message on this side.
type Decision string

const (
	DecisionSent     Decision = "sent"
	DecisionHeld     Decision = "held"
	DecisionApproved Decision = "approved"
	DecisionReleased Decision = "released"
	DecisionDenied   Decision = "denied"
	DecisionRejected Decision = "rejected"
)

// Consent is which standing permission of the user's let a message in unasked.
type Consent string

const (
	ConsentProject  Consent = "project"
	ConsentDispatch Consent = "dispatch"
	ConsentSwitch   Consent = "switch"
)

// AuditRecord is one line of the audit trail. It names the parties and says
// what was decided; it never carries the words of the message, only their
// length and a digest, so reading the trail reveals no conversation.
type AuditRecord struct {
	At       time.Time `json:"at"`
	Receiver string    `json:"receiver"`
	Sender   string    `json:"sender,omitempty"`
	ID       string    `json:"id,omitempty"`
	Decision Decision  `json:"decision"`
	Reason   Reason    `json:"reason,omitempty"`
	// Consent names the standing consent of the user's that released the message
	// without a card; empty for a decision the user made on the card.
	Consent Consent `json:"consent,omitempty"`
	Bytes   int     `json:"bytes,omitempty"`
	SHA256  string  `json:"sha256,omitempty"`
}

// Digest is the audit's stand-in for a message body.
func Digest(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

// Audit appends one record to the month's file under the peers directory.
func (d *Dir) Audit(r AuditRecord) error {
	if r.At.IsZero() {
		r.At = time.Now().UTC()
	}
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	dir := filepath.Join(d.root, "audit")
	if err := checkPrivate(dir, true); err != nil {
		return err
	}
	file := filepath.Join(dir, r.At.UTC().Format("2006-01")+".jsonl")
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	if err == nil {
		auditMonths.Delete(file)
	}
	return err
}

// AuditSince reads the audit records at or after since, oldest first. A month
// file that is missing or has a damaged line contributes what it can: the trail
// is a record of what happened, and a gap in it must not stop a reader.
func (d *Dir) AuditSince(since time.Time) []AuditRecord {
	var out []AuditRecord
	now := time.Now().UTC()
	for m := time.Date(since.UTC().Year(), since.UTC().Month(), 1, 0, 0, 0, 0, time.UTC); !m.After(now); m = m.AddDate(0, 1, 0) {
		for _, r := range readAuditMonth(filepath.Join(d.root, "audit", m.Format("2006-01")+".jsonl")) {
			if !r.At.Before(since) {
				out = append(out, r)
			}
		}
	}
	return out
}

// readAuditMonth parses one month once and reuses it while the file's stamp is
// unchanged. The returned slice is shared by every reader and must be treated
// as immutable.
func readAuditMonth(file string) []AuditRecord {
	info, err := os.Stat(file)
	if err != nil {
		return nil
	}
	if cached, ok := auditMonths.Load(file); ok {
		entry := cached.(auditMonth)
		if entry.size == info.Size() && entry.mod.Equal(info.ModTime()) {
			return entry.records
		}
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var records []AuditRecord
	for line := range bytes.SplitSeq(data, []byte{'\n'}) {
		var r AuditRecord
		if len(line) == 0 || json.Unmarshal(line, &r) != nil {
			continue
		}
		records = append(records, r)
	}
	auditMonths.Store(file, auditMonth{mod: info.ModTime(), size: info.Size(), records: records})
	return records
}
