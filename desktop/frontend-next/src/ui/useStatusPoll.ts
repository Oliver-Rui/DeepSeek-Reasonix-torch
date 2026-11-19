import { useEffect } from "react";

const STATUS_POLL_MS = 250;

// The next read is scheduled when the previous one settles, never on a clock: a
// kernel slower than the cadence would otherwise be asked again before it had
// answered, and the queue of unanswered reads is what starves the window's other
// requests of connections.
//
// A read that returns early — a hidden page, a pane that is not on screen — must
// still be woken when the reason goes away. The timer keeps running either way,
// so the wait is short, but a page that comes back to the front after a long
// spell in the background reads once immediately instead of waiting out the
// remainder of a tick.
export function useStatusPoll(on: boolean, read: () => Promise<unknown>, everyMs = STATUS_POLL_MS) {
  useEffect(() => {
    if (!on) return;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const tick = () => {
      const next = () => {
        if (!stopped) timer = setTimeout(tick, everyMs);
      };
      void read().then(next, next);
    };
    const wake = () => {
      if (stopped || document.hidden) return;
      clearTimeout(timer);
      tick();
    };
    tick();
    document.addEventListener("visibilitychange", wake);
    return () => {
      stopped = true;
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", wake);
    };
  }, [on, read, everyMs]);
}
