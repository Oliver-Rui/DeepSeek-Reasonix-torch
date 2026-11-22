import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

const css = readFileSync(new URL("./studio.css", import.meta.url), "utf8");
const narrow = readFileSync(new URL("./narrow.css", import.meta.url), "utf8");
const app = readFileSync(new URL("./app.css", import.meta.url), "utf8");
const phone = readFileSync(new URL("../design/phone.css", import.meta.url), "utf8");
const composerDesign = readFileSync(new URL("../design/composer.css", import.meta.url), "utf8");

// The conversation fold narrows composer controls onto a 30px icon grid. Two
// 14px icons need 34px, so a pill the fold collapses to 30px must also drop
// its chevron — the policy pill's fold already does, and the project pill's
// left the pair in place, overflowing its own rounded box by arithmetic.
const fold = (width: string) =>
  css.match(new RegExp(`@container conversation \\(max-width: ${width}\\) \\{[\\s\\S]*?\\n\\}`))?.[0] ?? "";
const squeeze = (width: string) =>
  narrow.match(new RegExp(`@container conversation \\(max-width: ${width}\\) \\{[\\s\\S]*?\\n\\}`))?.[0] ?? "";

describe("the conversation fold's 30px collapse", () => {
  it("hides the project pill's chevron before narrowing it to 30px", () => {
    const block = fold("720px");
    expect(block).toContain(".studio-branch-pop .studio-branch { width: 30px");
    expect(block).toMatch(
      /\.studio-branch-pop \.studio-branch[^{}]*\.studio-icon:last-child[\s\S]{0,40}\{[^}]*display:\s*none/,
    );
  });

  it("keeps the sibling collapses on the same pattern", () => {
    expect(fold("720px")).toMatch(
      /\.studio-mode-picker > \.studio-icon:last-child[^{}]*\{[^}]*display:\s*none/,
    );
    expect(fold("590px")).toMatch(
      /\.turntools \.policy \.studio-control-chevron[^{}]*\{[^}]*display:\s*none/,
    );
  });

  it("truncates the model name across the whole squeezed range", () => {
    // The name has to give way wherever the row squeezes (≤720), not only on
    // the wrapped phone line — between 280 and 400 the row overflows the
    // pane by exactly the width the truncation was missing.
    expect(squeeze("720px")).toMatch(
      /\.studio-model-control \.model-picker \.nm[^{}]*\{[^}]*overflow:\s*hidden/,
    );
    // .picker's flex:none (app.css) loads after the fold files, so the fold
    // restores shrink through a selector that outranks it regardless of
    // bundle order.
    expect(squeeze("720px")).toMatch(
      /\.turntools \.studio-model-group [^{]*\{[^}]*flex-shrink:\s*1/,
    );
  });

  it("sizes the phone drawer against the zoomed layout viewport", () => {
    expect(css).not.toMatch(/--rail-open:\s*min\(84vw/);
    expect(css).toMatch(/--rail-open:\s*min\(calc\(100vw \/ var\(--zoom, 1\) \* 0\.84\), 300px\)/);
  });

  it("caps the model menu inside narrow viewports", () => {
    // The menu is a fixed 300px card; on a 320px phone viewport it must yield
    // to the viewport the same way every other floating panel does.
    expect(composerDesign).not.toMatch(/\.composer-menu\.mode-menu \{ width: 300px/);
    expect(composerDesign).toMatch(
      /\.composer-menu\.mode-menu \{ width: min\(300px, calc\(100vw \/ var\(--zoom, 1\) - 24px\)\)/,
    );
  });

  it("clamps control tips inside the conversation container", () => {
    expect(css).toMatch(
      /\.studio-control-tip \{[^}]*max-width:\s*min\(220px, calc\(100cqw - 48px\)\)/,
    );
  });

  it("aligns the pill row with the composer card in the side fold", () => {
    expect(app).toMatch(/\[data-fold~="side"\] \.compose \{ width: calc\(100% - 28px\)/);
    // The pill row keeps its desktop formula otherwise; in the side fold it
    // must take the card's exact width or the folder pill floats off the
    // card's left edge.
    expect(app).toMatch(/\[data-fold~="side"\] \.composer-top \{ width: calc\(100% - 28px\)/);
  });

  it("lets the phone-fold model row shrink inside the design layer", () => {
    // The design layer's flex:none on the group and control outranks every
    // legacy squeeze rule by layer order, so the phone fold — where the row
    // squeezes onto one line — must carry the shrink chain in the design
    // layer itself.
    expect(phone).toMatch(
      /\[data-fold~="phone"\] \.compose \.turntools > \.studio-model-group \{[^}]*flex:\s*0 1 auto[^}]*min-width:\s*0/,
    );
    expect(phone).toMatch(
      /\[data-fold~="phone"\] \.compose \.turntools \.studio-model-control \{[^}]*flex:\s*0 1 auto/,
    );
  });
});
