import type { UIAction } from "./actions";

// The sidebar tree and the openings it offers. Reached from the workspace tree
// and from the remote host tree, which are two surfaces for one intent; opening
// an existing session is not starting a new one, and the host asked differs.
// A drag or a menu move is one kernel place, so its proof is the order itself.
export const TREE_ACTIONS: UIAction[] = [
  { id: "session.open", kind: "kernel-mutation", target: "none", proof: "authority-effect" },
  { id: "remote.open", kind: "kernel-mutation", target: "none", proof: "authority-effect" },
  // Asking a machine whether it answers. Nothing on either side changes, and
  // asking twice is the point when the first answer was "not yet".
  { id: "remote.probe", kind: "repeatable", target: "entity", proof: "authority-effect" },
  { id: "remote.hide", kind: "view", target: "entity", proof: "interaction" },
  { id: "remote.more", kind: "view", target: "entity", proof: "interaction" },
  { id: "remote.edit", kind: "view", target: "entity", proof: "interaction" },
  { id: "remote.remove-ask", kind: "view", target: "entity", proof: "interaction" },
  { id: "remotes.add-all", kind: "kernel-mutation", target: "none", proof: "interaction" },
  { id: "remote.enable", kind: "view", target: "entity", proof: "interaction" },
  { id: "remote.read", kind: "repeatable", target: "entity", proof: "authority-effect" },
  { id: "workspace.add", kind: "kernel-mutation", target: "none", proof: "authority-effect" },
  { id: "workspace.fold-all", kind: "view", target: "none", proof: "interaction" },
  { id: "workspace.fold", kind: "view", target: "none", proof: "interaction" },
  { id: "workspace.rename-start", kind: "view", target: "entity", proof: "interaction" },
  { id: "workspace.rename", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "workspace.rename-close", kind: "view", target: "none", proof: "interaction" },
  { id: "workspace.pin", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "workspace.archive-sessions", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "workspace.copy-dir", kind: "view", target: "entity", proof: "interaction" },
  { id: "workspace.add-cancel", kind: "view", target: "none", proof: "interaction" },
  { id: "root-missing.reselect", kind: "view", target: "none", proof: "interaction" },
  { id: "workspace.move", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "workspace.reorder", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "session.reorder", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "session.move", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "session.kept", kind: "view", target: "entity", proof: "interaction" },
  { id: "session.move-to-start", kind: "view", target: "entity", proof: "interaction" },
  { id: "session.move-to", kind: "kernel-mutation", target: "entity", proof: "authority-effect" },
  { id: "session.move-to-cancel", kind: "view", target: "none", proof: "interaction" },
  { id: "session.move-to-target", kind: "view", target: "none", proof: "interaction" },
  { id: "session.move-to-add", kind: "view", target: "none", proof: "interaction" },
];
