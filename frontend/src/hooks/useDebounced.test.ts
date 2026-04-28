import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useDebounced } from "./useDebounced";

describe("useDebounced", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  test("does not update before delay", () => {
    const { result, rerender } = renderHook(({ value }) => useDebounced(value, 100), {
      initialProps: { value: "a" },
    });
    rerender({ value: "ab" });
    expect(result.current).toBe("a");
    act(() => {
      vi.advanceTimersByTime(50);
    });
    expect(result.current).toBe("a");
  });

  test("updates after delay", () => {
    const { result, rerender } = renderHook(({ value }) => useDebounced(value, 100), {
      initialProps: { value: "a" },
    });
    rerender({ value: "ab" });
    act(() => {
      vi.advanceTimersByTime(120);
    });
    expect(result.current).toBe("ab");
  });

  test("rapid changes only emit the last value", () => {
    const { result, rerender } = renderHook(({ value }) => useDebounced(value, 100), {
      initialProps: { value: "" },
    });
    rerender({ value: "a" });
    act(() => {
      vi.advanceTimersByTime(40);
    });
    rerender({ value: "ab" });
    act(() => {
      vi.advanceTimersByTime(40);
    });
    rerender({ value: "abc" });
    act(() => {
      vi.advanceTimersByTime(120);
    });
    expect(result.current).toBe("abc");
  });
});
