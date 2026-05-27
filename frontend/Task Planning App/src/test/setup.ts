import "@testing-library/jest-dom";

// Polyfill ResizeObserver: jsdom не предоставляет его, а radix-ui (ScrollArea и др.)
// вызывает его на mount.
if (typeof globalThis.ResizeObserver === "undefined") {
    globalThis.ResizeObserver = class {
        observe() {}
        unobserve() {}
        disconnect() {}
    } as unknown as typeof ResizeObserver;
}

class MemoryStorage implements Storage {
    private data = new Map<string, string>();
    get length() {
        return this.data.size;
    }
    clear() {
        this.data.clear();
    }
    getItem(key: string) {
        return this.data.has(key) ? (this.data.get(key) as string) : null;
    }
    key(index: number) {
        return Array.from(this.data.keys())[index] ?? null;
    }
    removeItem(key: string) {
        this.data.delete(key);
    }
    setItem(key: string, value: string) {
        this.data.set(key, String(value));
    }
}

function installStorage(name: "localStorage" | "sessionStorage") {
    const current = (globalThis as unknown as Record<string, unknown>)[name];
    if (current && typeof (current as Storage).setItem === "function") {
        return;
    }
    const storage = new MemoryStorage();
    Object.defineProperty(globalThis, name, {
        value: storage,
        configurable: true,
        writable: true,
    });
    if (typeof window !== "undefined") {
        Object.defineProperty(window, name, {
            value: storage,
            configurable: true,
            writable: true,
        });
    }
}

installStorage("localStorage");
installStorage("sessionStorage");
