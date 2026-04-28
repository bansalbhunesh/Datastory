import { afterEach, describe, expect, test, vi } from "vitest";
import { fetchIncidents } from "./incidents";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("fetchIncidents", () => {
  test("returns [] for blank tableFQN without hitting the network", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    const out = await fetchIncidents("   ");
    expect(out).toEqual([]);
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  test("returns entries on success", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          incidents: [
            {
              id: "1",
              createdAt: 100,
              tableFQN: "a",
              severity: "LOW",
              source: "deterministic",
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    const out = await fetchIncidents("a");
    expect(out).toHaveLength(1);
    expect(out[0].id).toBe("1");
  });

  test("returns [] when server errors", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("nope", { status: 500 }));
    const out = await fetchIncidents("a");
    expect(out).toEqual([]);
  });

  test("returns [] when network throws", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("offline"));
    const out = await fetchIncidents("a");
    expect(out).toEqual([]);
  });

  test("returns [] when server returns malformed JSON", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response("not-json", { status: 200, headers: { "Content-Type": "application/json" } }),
    );
    const out = await fetchIncidents("a");
    expect(out).toEqual([]);
  });
});
