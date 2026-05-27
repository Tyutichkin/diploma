declare global {
    interface Window {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        ymaps: any;
    }
}

let loadPromise: Promise<void> | null = null;

export function loadYandexMaps(): Promise<void> {
    if (window.ymaps && typeof window.ymaps.Map === "function") {
        return Promise.resolve();
    }

    if (loadPromise) return loadPromise;

    const apiKey =
        (import.meta.env.VITE_YANDEX_GEOCODER_KEY as string | undefined) ||
        (import.meta.env.VITE_YANDEX_MAPS_KEY as string | undefined);

    if (!apiKey) {
        return Promise.reject(
            new Error("VITE_YANDEX_GEOCODER_KEY не задан в .env.local"),
        );
    }

    loadPromise = new Promise<void>((resolve, reject) => {
        const existing = document.querySelector(
            'script[src^="https://api-maps.yandex.ru"]',
        );
        if (existing) {
            window.ymaps.ready(resolve);
            return;
        }

        const script = document.createElement("script");
        script.src = `https://api-maps.yandex.ru/2.1/?apikey=${encodeURIComponent(apiKey)}&lang=ru_RU`;
        script.type = "text/javascript";
        script.async = true;
        script.onload = () => window.ymaps.ready(resolve);
        script.onerror = () => {
            loadPromise = null;
            reject(
                new Error(
                    "Не удалось загрузить Яндекс Карты. Проверьте API-ключ.",
                ),
            );
        };
        document.head.appendChild(script);
    });

    return loadPromise;
}
