import { describe, expect, test } from "vitest";
import { __test } from "./report";

const { normalizeReport } = __test;

describe("normalizeReport", () => {
  test("rejects non-object input", () => {
    expect(() => normalizeReport(null)).toThrow();
    expect(() => normalizeReport(42)).toThrow();
    expect(() => normalizeReport("oops")).toThrow();
  });

  test("rejects missing tableFQN/markdown", () => {
    expect(() => normalizeReport({ tableFQN: "", markdown: "x" })).toThrow();
    expect(() => normalizeReport({ tableFQN: "x", markdown: "" })).toThrow();
  });

  test("normalizes 'llm' source to 'claude'", () => {
    const r = normalizeReport({
      tableFQN: "a.b.c",
      markdown: "# x",
      source: "llm",
      lineage: { focal: "a.b.c", upstream: [], downstream: [], upstreamRaw: 0, downstreamRaw: 0 },
      failedTests: [],
    });
    expect(r.source).toBe("claude");
  });

  test("falls back to deterministic for unknown source", () => {
    const r = normalizeReport({
      tableFQN: "a.b.c",
      markdown: "# x",
      source: "wat",
      lineage: { focal: "a.b.c", upstream: [], downstream: [], upstreamRaw: 0, downstreamRaw: 0 },
      failedTests: [],
    });
    expect(r.source).toBe("deterministic");
  });

  test("uppercases and validates severity", () => {
    const r = normalizeReport({
      tableFQN: "a.b.c",
      markdown: "# x",
      severity: "high",
      source: "claude",
      lineage: { focal: "", upstream: [], downstream: [], upstreamRaw: 0, downstreamRaw: 0 },
      failedTests: [],
    });
    expect(r.severity).toBe("HIGH");
  });

  test("falls back to UNKNOWN for invalid severity", () => {
    const r = normalizeReport({
      tableFQN: "a.b.c",
      markdown: "# x",
      severity: "WARN",
      source: "claude",
      lineage: { focal: "", upstream: [], downstream: [], upstreamRaw: 0, downstreamRaw: 0 },
      failedTests: [],
    });
    expect(r.severity).toBe("UNKNOWN");
  });

  test("filters non-string entries from string arrays", () => {
    const r = normalizeReport({
      tableFQN: "a.b.c",
      markdown: "# x",
      source: "claude",
      rootCauses: ["ok", 5, null, "also"],
      lineage: { focal: "", upstream: ["u", 7], downstream: [null, "d"], upstreamRaw: 0, downstreamRaw: 0 },
      failedTests: [],
    });
    expect(r.rootCauses).toEqual(["ok", "also"]);
    expect(r.lineage.upstream).toEqual(["u"]);
    expect(r.lineage.downstream).toEqual(["d"]);
  });

  test("drops malformed failedTest entries (no name)", () => {
    const r = normalizeReport({
      tableFQN: "a.b.c",
      markdown: "# x",
      source: "claude",
      lineage: { focal: "", upstream: [], downstream: [], upstreamRaw: 0, downstreamRaw: 0 },
      failedTests: [
        { name: "ok", status: "Failed", result: "r" },
        { name: "", status: "Failed" },
        "garbage",
      ],
    });
    expect(r.failedTests).toHaveLength(1);
    expect(r.failedTests[0].name).toBe("ok");
  });

  test("provides safe lineage default when missing", () => {
    const r = normalizeReport({
      tableFQN: "a.b.c",
      markdown: "# x",
      source: "claude",
      failedTests: [],
    });
    expect(r.lineage.focal).toBe("");
    expect(r.lineage.upstream).toEqual([]);
    expect(r.lineage.downstream).toEqual([]);
  });
});
