import "@testing-library/jest-dom/vitest";

// Polyfill for components that touch clipboard during tests.
if (!("clipboard" in navigator)) {
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: () => Promise.resolve() },
    configurable: true,
  });
}

// jsdom doesn't implement scrollIntoView; some libs (recharts/reactflow) call it.
if (typeof window !== "undefined" && !Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {};
}
