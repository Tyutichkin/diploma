import '@testing-library/jest-dom';

// Polyfill ResizeObserver: jsdom не предоставляет его, а radix-ui (ScrollArea и др.)
// вызывает его на mount.
if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
}
