import { useEffect, useMemo, useRef, useState } from "react";
import type { AgentPort, PeersState } from "../port/port";
import { singleFlight } from "./singleflight";

const EMPTY: PeersState = { enabled: false, peers: [], held: [] };

export function usePeers(port: AgentPort | null, everyMs: number, active = true) {
  const [state, setState] = useState<PeersState>(EMPTY);
  const [failed, setFailed] = useState(false);
  const alive = useRef(true);
  const refresh = useMemo(() => singleFlight(async () => {
    if (!port) return;
    try {
      const next = await port.peers();
      if (alive.current) {
        setState(next);
        setFailed(false);
      }
    } catch {
      if (alive.current) setFailed(true);
    }
  }), [port]);
  useEffect(() => {
    if (!active) return;
    alive.current = true;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const tick = () => {
      if (typeof document !== "undefined" && document.hidden) {
        if (everyMs > 0) timer = setTimeout(tick, everyMs);
        return;
      }
      void refresh().then(() => {
        if (!stopped && everyMs > 0) timer = setTimeout(tick, everyMs);
      });
    };
    tick();
    return () => {
      stopped = true;
      clearTimeout(timer);
      alive.current = false;
    };
  }, [refresh, everyMs, active]);
  return { state, failed, refresh };
}
