import type { UIAction } from "./actions";

// The two dispatch ceilings on the session cross-chat page. The input commits on
// blur the way the reference-size row does — one Enter, one save, and what the
// kernel keeps comes back to the box — so the canonical change is the proof.
export const PEERS_ACTIONS: UIAction[] = [
  { id: "peers.dispatch-depth", kind: "kernel-mutation", target: "none", proof: "authority-effect" },
  { id: "peers.dispatched-sessions", kind: "kernel-mutation", target: "none", proof: "authority-effect" },
  // The outgoing cross-project seg on the switches block: it saves the whole
  // answer through the same settings POST as the switches beside it.
  { id: "peers.cross-project", kind: "kernel-mutation", target: "none", proof: "authority-effect" },
  // The dispatch dialog: one task to several sessions, each under its own role.
  // The fan-out writes through the hub, so the proof is the canonical effect.
  { id: "session.dispatch-start", kind: "view", target: "entity", proof: "interaction" },
  { id: "session.dispatch", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "session.dispatch-mode", kind: "view", target: "entity", proof: "interaction" },
  { id: "session.dispatch-target", kind: "view", target: "entity", proof: "interaction" },
  { id: "session.dispatch-project", kind: "view", target: "entity", proof: "interaction" },
  { id: "session.dispatch-role", kind: "view", target: "entity", proof: "interaction" },
  { id: "session.dispatch-instructions", kind: "view", target: "entity", proof: "interaction" },
  { id: "session.dispatch-add", kind: "view", target: "none", proof: "interaction" },
  { id: "session.dispatch-remove", kind: "view", target: "none", proof: "interaction" },
  { id: "session.dispatch-done", kind: "view", target: "none", proof: "interaction" },
];
