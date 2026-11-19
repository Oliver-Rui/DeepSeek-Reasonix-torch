package provider

// DecisionReceipt is durable, provider-excluded evidence of a user-owned
// approval decision. It intentionally contains only bounded labels and the
// outcome, never free-form guidance or provider-visible content.
type DecisionReceipt struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Tool    string `json:"tool,omitempty"`
	Subject string `json:"subject,omitempty"`
	Outcome string `json:"outcome"`
	// Via names the paired device the decision was made on; nil is the window.
	Via *Via `json:"via,omitempty"`
}

// Via is the paired device a person acted from. Declared by the host that
// authenticated the device, never read off the input: a message cannot say
// where it was typed.
type Via struct {
	// Device is the pairing's id, stable while the device stays paired.
	Device string `json:"device"`
	// Ordinal is the number the window listed the device under when it acted,
	// the 设备 N both sides showed at the time.
	Ordinal int `json:"ordinal"`
}

// PeerSource names the other session a user-role message came from. Declared
// by the host that delivered it, never read off the body: a message cannot say
// who sent it, and the wording of a body is data the sender controls. Local UI
// metadata, stripped before every provider request like Via.
type PeerSource struct {
	// SessionID is the sender's peer key, which is what a client uses to open
	// the conversation the message came from.
	SessionID string `json:"sessionId"`
	// Title is the sender's name when the host knew it at delivery time.
	Title string `json:"title,omitempty"`
	// Path is the sender's transcript, so the label survives the peer going
	// quiet and its live record disappearing.
	Path string `json:"path,omitempty"`
	// Reply marks a message a dispatched member sent back, rather than a
	// dispatch the lead handed out.
	Reply bool `json:"reply,omitempty"`
}
