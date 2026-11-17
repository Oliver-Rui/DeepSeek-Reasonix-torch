package control

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"reasonix/internal/state/peermail"
	"reasonix/internal/state/sessionstore"
)

// NewPairSecret makes the secret one dispatch pairing signs its messages with.
// It lives in the two controllers' memory and is never written to the spool.
// The host creates it so both ends of a dispatch-to-existing pair share one.
func NewPairSecret() ([]byte, error) {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	return secret, err
}

// DispatchToSessionKey hands a task to an already-listed session by its peer
// key, pairing it as a dispatch target and writing the message straight to that
// session's spool. Offline targets take it when they next ingest; the dispatch
// pairing, not the receiver's inbound card, is the consent. A secret the host
// keeps in memory signs the pair so a copied message cannot be replayed.
func (c *Controller) DispatchToSessionKey(key, role, text string, secret []byte) (PeerReceipt, error) {
	p := c.peers
	dir := p.dirOrNil()
	if dir == nil {
		return PeerReceipt{}, ErrPeersDisabled
	}
	if len(secret) == 0 {
		return PeerReceipt{}, errors.New("dispatch: no pairing secret")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return PeerReceipt{}, ErrPeerEmpty
	}
	p.mu.Lock()
	self, path := p.key, c.SessionPath()
	p.mu.Unlock()
	if self == "" {
		return PeerReceipt{}, ErrPeersDisabled
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return PeerReceipt{}, ErrPeerNotFound
	}
	if key == self {
		return PeerReceipt{}, ErrPeerSelf
	}
	if !c.peerSendAllowed() {
		return PeerReceipt{}, ErrPeerRateLimited
	}
	c.NoteDispatchedSession(key, secret, role)
	var token [8]byte
	_, _ = rand.Read(token[:])
	msg := peermail.Message{
		ID: "m-" + hex.EncodeToString(token[:]), SentAt: time.Now().UTC(), Body: text,
		From: peermail.Party{SessionID: self, Title: peerAttribute(sessionstore.BranchID(path)), Project: peerAttribute(baseName(c.WorkspaceRoot()))},
		To:   peermail.Party{SessionID: key},
	}
	c.signDispatch(&msg)
	if err := dir.Send(msg); err != nil {
		return PeerReceipt{}, err
	}
	_ = dir.Audit(peermail.AuditRecord{Receiver: key, Sender: self, ID: msg.ID, Decision: peermail.DecisionSent, Bytes: len(msg.Body), SHA256: peermail.Digest(msg.Body)})
	return PeerReceipt{ID: msg.ID, To: PeerView{Key: key}}, nil
}
