import { useCallback, useEffect, useRef, useState } from "react";
import type { DispatchBoardRow, HubPort } from "../port/hub";

/** The machine-wide session-collaboration board. The hub reads the durable
 *  peer trail, so a closed session still has its row; polling mirrors usePeers
 *  and pauses while the window is hidden. */
export function useDispatchBoard(hub: HubPort | null, everyMs = 3000) {
  const [rows, setRows] = useState<DispatchBoardRow[]>([]);
  const [failed, setFailed] = useState(false);
  const alive = useRef(true);
  const refresh = useCallback(async () => {
    if (!hub) return;
    try {
      const next = await hub.dispatchBoard();
      if (alive.current) {
        setRows(next);
        setFailed(false);
      }
    } catch {
      if (alive.current) setFailed(true);
    }
  }, [hub]);
  useEffect(() => {
    alive.current = true;
    void refresh();
    const timer = everyMs > 0
      ? setInterval(() => {
          if (typeof document === "undefined" || !document.hidden) void refresh();
        }, everyMs)
      : null;
    return () => {
      alive.current = false;
      if (timer) clearInterval(timer);
    };
  }, [refresh, everyMs]);
  return { rows, failed, refresh };
}
