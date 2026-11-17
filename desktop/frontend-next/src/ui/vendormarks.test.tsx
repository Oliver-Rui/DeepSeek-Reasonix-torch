import { describe, expect, it } from "vitest";
import { markForModelId } from "./vendormarks";

// Some relays name their models in private shorthand; the mark still reads the
// family the name stands for, and the list says the model's own name, with or
// without a provider prefix in front.
describe("the model mark reads through a relay's private names", () => {
  it.each<[string, string]>([
    ["fast", "qwen"],
    ["qoder/fast", "qwen"],
    ["qmodel_latest", "qwen"],
    ["qmodel_38max", "qwen"],
    ["qfmodel", "qwen"],
    ["auto/qfmodel", "qwen"],
    ["dmodel", "deepseek"],
    ["dfmodel", "deepseek"],
    ["gmodel", "zhipu"],
    ["gfmodel", "zhipu"],
    ["kmodel_latest", "kimi"],
    ["kmodel", "kimi"],
    ["jmodel", "stepfun"],
    ["mmodel", "minimax"],
    ["mmmodel", "minimax"],
  ])("reads %s as %s", (id, mark) => {
    expect(markForModelId(id)).toBe(mark);
  });

  it("still reads a self-naming id by its own keywords", () => {
    expect(markForModelId("deepseek-chat")).toBe("deepseek");
    expect(markForModelId("group/glm-5.3-flash")).toBe("zhipu");
    // A name no table knows reads as nothing, leaving the door's own mark.
    expect(markForModelId("auto/m/model")).toBeUndefined();
  });

  // The Hunyuan family ships as hy4-flash on the gateway, so an id that never
  // spells the family out still has to reach its mark.
  it.each(["group/hy4-flash", "hy4-flash", "group/hy-7", "hunyuan-turbo", "混元-4"])("reads %s as hunyuan", (id) => {
    expect(markForModelId(id)).toBe("hunyuan");
  });

  it("does not read hy inside another word as the family", () => {
    expect(markForModelId("rhythm")).toBeUndefined();
    expect(markForModelId("shy-model")).toBeUndefined();
  });
});
