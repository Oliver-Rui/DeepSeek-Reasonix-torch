import { t } from "../i18n";

const DAY = 86_400_000;

/** The coarsest true unit for a recent timestamp. The session list and the
 *  collaboration board share it so one row of either never reads two ages. */
export function ageWord(at: number, now: number): string {
  const gap = now - at;
  if (gap < 60_000) return t("common.just_now");
  if (gap < 3_600_000) return t("common.n_min", { n: Math.floor(gap / 60_000) });
  const day = (ms: number) => new Date(ms).setHours(0, 0, 0, 0);
  if (day(now) === day(at)) return t("common.n_h", { n: Math.floor(gap / 3_600_000) });
  if (day(now) - day(at) === DAY) return t("common.yesterday");
  const d = new Date(at);
  return `${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
