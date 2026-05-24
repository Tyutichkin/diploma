/* eslint-disable @typescript-eslint/no-explicit-any */
import { Task } from "../../types/task";

const MARKER_SIZE = 32;
const MARKER_RADIUS = MARKER_SIZE / 2;
// Радиус кликабельной зоны чуть больше визуального круга: компенсирует
// мелкие промахи по рамке/тени и сужает зазор между маркерами в стопке.
const MARKER_HITBOX_RADIUS = 19;
const MARKER_FONT_SIZE = 14;
const MARKER_BORDER_WIDTH = 2;
const ANCHOR_SIZE = 6;
const ANCHOR_RADIUS = ANCHOR_SIZE / 2;

const Z_ANCHOR = 650;
const Z_MARKER = 700;
const Z_STACK_MARKER = 720;
const Z_MARKER_HOVER = 800;

interface CreateTaskMarkerArgs {
    task: Task;
    number: number;
    iconOffset: [number, number];
    isStack: boolean;
}

// SVG-кружок белый с синей рамкой как data URI. Используется в стандартном
// imageWithContent layout — Yandex сам обслуживает позиционирование и хит-тест
// по <img>, поэтому события не теряются между DOM и events-pane карты.
const MARKER_SVG_DATA_URI = (() => {
    const radius = MARKER_RADIUS - MARKER_BORDER_WIDTH / 2;
    const svg =
        `<svg xmlns="http://www.w3.org/2000/svg" width="${MARKER_SIZE}" height="${MARKER_SIZE}" viewBox="0 0 ${MARKER_SIZE} ${MARKER_SIZE}">` +
        `<circle cx="${MARKER_RADIUS}" cy="${MARKER_RADIUS}" r="${radius}" fill="#ffffff" stroke="#2563eb" stroke-width="${MARKER_BORDER_WIDTH}"/>` +
        `</svg>`;
    return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`;
})();

let cachedContentLayout: any = null;

function getContentLayout(): any {
    if (cachedContentLayout) return cachedContentLayout;
    // 32×32 flex-container, чтобы цифра рисовалась ровно поверх SVG-кружка.
    // pointer-events:none, чтобы клики по цифре не съедались DOM-листенерами
    // контента и попадали в hotspot <img>, который и должен их получать.
    const template =
        `<div style="box-sizing:border-box;display:flex;align-items:center;justify-content:center;` +
        `width:${MARKER_SIZE}px;height:${MARKER_SIZE}px;color:#2563eb;font-weight:700;` +
        `font-size:${MARKER_FONT_SIZE}px;line-height:1;user-select:none;pointer-events:none;` +
        `font-family:inherit;">` +
        `$[properties.iconContent]` +
        `</div>`;
    cachedContentLayout = window.ymaps.templateLayoutFactory.createClass(template);
    return cachedContentLayout;
}

export function createTaskMarker({
    task,
    number,
    iconOffset,
    isStack,
}: CreateTaskMarkerArgs): any {
    const balloonHeader = `#${number} ${task.title}`;
    const balloonBody = [
        task.address ? `<div style="color:#555">${task.address}</div>` : "",
        task.duration != null
            ? `<div style="color:#555;margin-top:2px">Длительность: ${task.duration} мин</div>`
            : "",
        task.windowStartDate ||
        task.windowStartTime ||
        task.windowEndDate ||
        task.windowEndTime
            ? `<div style="color:#555">Окно: ${[task.windowStartDate, task.windowStartTime].filter(Boolean).join(" ") || "—"} – ${[task.windowEndDate, task.windowEndTime].filter(Boolean).join(" ") || "—"}</div>`
            : "",
    ]
        .filter(Boolean)
        .join("");

    // Центр hotspot-зоны совпадает с центром визуального круга: iconImageOffset
    // задаёт top-left, прибавляем MARKER_RADIUS для центра.
    const shapeCenter: [number, number] = [
        iconOffset[0] + MARKER_RADIUS,
        iconOffset[1] + MARKER_RADIUS,
    ];

    return new window.ymaps.Placemark(
        [task.latitude!, task.longitude!],
        {
            iconContent: String(number),
            balloonContentHeader: balloonHeader,
            balloonContentBody: balloonBody,
            hintContent: task.title,
        },
        {
            // Стандартный layout: <img> для кружка + custom layout для цифры.
            // Никакого кастомного HTML-контейнера для самого маркера — ymaps
            // сам ставит <img> по iconImageOffset и принимает события без
            // зависимости от DOM-pointer-events.
            iconLayout: "default#imageWithContent",
            iconImageHref: MARKER_SVG_DATA_URI,
            iconImageSize: [MARKER_SIZE, MARKER_SIZE],
            iconImageOffset: iconOffset,
            iconContentLayout: getContentLayout(),
            iconContentOffset: [0, 0],
            iconShape: {
                type: "Circle",
                coordinates: shapeCenter,
                radius: MARKER_HITBOX_RADIUS,
            },
            // opaque: события полностью гасятся на placemark, карта не получает
            // mousedown и не интерпретирует мелкое движение как drag, dblclick
            // не уходит в dblClickZoom behavior карты.
            interactivityModel: "default#opaque",
            openBalloonOnClick: true,
            draggable: false,
            cursor: "pointer",
            zIndex: isStack ? Z_STACK_MARKER : Z_MARKER,
            zIndexHover: Z_MARKER_HOVER,
        },
    );
}

interface CreateStackAnchorArgs {
    lat: number;
    lon: number;
    lineHeightPx: number;
}

const anchorLayoutCache = new Map<number, any>();

function getAnchorLayout(lineHeightPx: number): any {
    let layout = anchorLayoutCache.get(lineHeightPx);
    if (layout) return layout;
    layout = window.ymaps.templateLayoutFactory.createClass(
        [
            '<div style="position:relative;width:6px;height:6px;">',
            '<div style="position:absolute;left:50%;bottom:3px;transform:translateX(-50%);',
            `width:1px;height:${lineHeightPx}px;background:#9ca3af;"></div>`,
            '<div style="position:absolute;inset:0;border-radius:50%;background:#9ca3af;',
            'box-shadow:0 0 0 1px #ffffff;"></div>',
            "</div>",
        ].join(""),
    );
    anchorLayoutCache.set(lineHeightPx, layout);
    return layout;
}

export function createStackAnchor({
    lat,
    lon,
    lineHeightPx,
}: CreateStackAnchorArgs): any {
    return new window.ymaps.Placemark(
        [lat, lon],
        {},
        {
            iconLayout: getAnchorLayout(lineHeightPx),
            iconShape: {
                type: "Circle",
                coordinates: [0, 0],
                radius: ANCHOR_RADIUS,
            },
            iconOffset: [-ANCHOR_RADIUS, -ANCHOR_RADIUS],
            zIndex: Z_ANCHOR,
            hasBalloon: false,
            hasHint: false,
        },
    );
}
