package serve

import "reasonix/internal/contract/provider"

// peerSourceWire is the label a transcript row carries when another session
// relayed the message. It is the host's account, written when the message
// landed; a client renders it and never re-derives it from the body.
type peerSourceWire struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title,omitempty"`
	Path      string `json:"path,omitempty"`
	Reply     bool   `json:"reply,omitempty"`
}

func peerSourceView(src *provider.PeerSource) *peerSourceWire {
	if src == nil || src.SessionID == "" {
		return nil
	}
	return &peerSourceWire{SessionID: src.SessionID, Title: src.Title, Path: src.Path, Reply: src.Reply}
}
