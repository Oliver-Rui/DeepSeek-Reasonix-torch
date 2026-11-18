package control

import "testing"

func TestDispatchGrantRunsTheCrewUnattended(t *testing.T) {
	for _, tc := range []struct{ set, want string }{
		{ToolApprovalAsk, ToolApprovalAuto},
		{ToolApprovalAuto, ToolApprovalAuto},
		{ToolApprovalYolo, ToolApprovalYolo},
		{"dontask", "dontAsk"},
	} {
		c := New(Options{})
		c.SetToolApprovalMode(tc.set)
		if got := c.DispatchGrant().Approval; got != tc.want {
			t.Errorf("lead %q passes %q, want %q", tc.set, got, tc.want)
		}
		c.Close()
	}
}

func TestDispatchedMemberDoesNotAskItsLeadToApprove(t *testing.T) {
	// A lead in ask mode must not hand that posture to a crew member: the lead
	// cannot see the member's approval card, so the member would stall and the
	// window would keep announcing a card nobody there can answer.
	lead := New(Options{})
	lead.SetToolApprovalMode(ToolApprovalAsk)
	grant := lead.DispatchGrant()
	member := New(Options{})
	member.AdoptSessionGrant(grant)
	if got := member.ToolApprovalMode(); got == ToolApprovalAsk {
		t.Fatalf("member inherited ask posture; it would stall on every tool")
	}
	defer member.Close()
	defer lead.Close()
}
