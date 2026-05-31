# -*- coding: utf-8 -*-
"""Примитивы блок-схемы по ГОСТ 19.701-90 для matplotlib.

Общий модуль фигур для блок-схемы алгоритма NNH-TW
(nnh_tw_flowchart.py). Все фигуры белые с чёрным контуром,
поток сверху вниз, вход в блок — сверху, выход — снизу.
"""
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

LW = 1.3
EC = 'black'
FC = 'white'
LINESPACING = 1.3

_meas_fig = None


def text_size(text, fs, dpi=100):
    """Размер блока текста в дюймах (= единицы данных при scale 1 дюйм/единица).

    Измеряет реальный габарит через скрытую фигуру тем же шрифтом и
    межстрочным интервалом, что и при отрисовке (_label)."""
    global _meas_fig
    if _meas_fig is None:
        _meas_fig = plt.figure(dpi=dpi)
    t = _meas_fig.text(0.5, 0.5, text, fontsize=fs, ha='center', va='center',
                       multialignment='center', linespacing=LINESPACING)
    _meas_fig.canvas.draw()
    bb = t.get_window_extent(renderer=_meas_fig.canvas.get_renderer())
    t.remove()
    return bb.width / dpi, bb.height / dpi


def _label(ax, cx, cy, text, fs, bold=False):
    ax.text(cx, cy, text, ha='center', va='center', fontsize=fs,
            fontweight='bold' if bold else 'normal',
            zorder=4, multialignment='center', linespacing=LINESPACING)


def terminal(ax, cx, cy, w, h, text, fs=9):
    """Терминатор (начало/конец) — овал."""
    ax.add_patch(mpatches.FancyBboxPatch(
        (cx - w / 2 + h / 2, cy - h / 2), w - h, h,
        boxstyle=f'round,pad=0,rounding_size={h/2}',
        fc=FC, ec=EC, lw=LW, zorder=3))
    _label(ax, cx, cy, text, fs, bold=True)


def process(ax, cx, cy, w, h, text, fs=8.5):
    """Процесс — прямоугольник."""
    ax.add_patch(plt.Rectangle((cx - w / 2, cy - h / 2), w, h,
                               fc=FC, ec=EC, lw=LW, zorder=3))
    _label(ax, cx, cy, text, fs)


def predefined(ax, cx, cy, w, h, text, fs=8.5):
    """Предопределённый процесс (подпрограмма) — прямоугольник с боковыми линиями."""
    ax.add_patch(plt.Rectangle((cx - w / 2, cy - h / 2), w, h,
                               fc=FC, ec=EC, lw=LW, zorder=3))
    d = min(0.22, w * 0.06)
    ax.plot([cx - w / 2 + d, cx - w / 2 + d], [cy - h / 2, cy + h / 2],
            color=EC, lw=LW, zorder=4)
    ax.plot([cx + w / 2 - d, cx + w / 2 - d], [cy - h / 2, cy + h / 2],
            color=EC, lw=LW, zorder=4)
    _label(ax, cx, cy, text, fs)


def decision(ax, cx, cy, w, h, text, fs=8.5):
    """Решение — ромб."""
    xs = [cx, cx + w / 2, cx, cx - w / 2, cx]
    ys = [cy + h / 2, cy, cy - h / 2, cy, cy + h / 2]
    ax.add_patch(mpatches.Polygon(list(zip(xs, ys)), closed=True,
                                  fc=FC, ec=EC, lw=LW, zorder=3))
    _label(ax, cx, cy, text, fs)


def io_block(ax, cx, cy, w, h, text, fs=8.5):
    """Данные (ввод/вывод) — параллелограмм."""
    d = h * 0.42
    xs = [cx - w / 2 + d, cx + w / 2 + d, cx + w / 2 - d, cx - w / 2 - d]
    ys = [cy + h / 2, cy + h / 2, cy - h / 2, cy - h / 2]
    ax.add_patch(mpatches.Polygon(list(zip(xs, ys)), closed=True,
                                  fc=FC, ec=EC, lw=LW, zorder=3))
    _label(ax, cx, cy, text, fs)


def prepare(ax, cx, cy, w, h, text, fs=8.5):
    """Подготовка (инициализация цикла/счётчика) — шестиугольник."""
    d = h * 0.55
    xs = [cx - w / 2 + d, cx + w / 2 - d, cx + w / 2,
          cx + w / 2 - d, cx - w / 2 + d, cx - w / 2]
    ys = [cy + h / 2, cy + h / 2, cy, cy - h / 2, cy - h / 2, cy]
    ax.add_patch(mpatches.Polygon(list(zip(xs, ys)), closed=True,
                                  fc=FC, ec=EC, lw=LW, zorder=3))
    _label(ax, cx, cy, text, fs)


def connector(ax, cx, cy, r, text, fs=8.5):
    """Соединитель — окружность."""
    ax.add_patch(plt.Circle((cx, cy), r, fc=FC, ec=EC, lw=LW, zorder=3))
    _label(ax, cx, cy, text, fs, bold=True)


def loop_start(ax, cx, cy, w, h, text, fs=8.5):
    """Граница цикла (начало) — прямоугольник со срезанными верхними углами."""
    c = min(0.32, h * 0.42)
    xs = [cx - w / 2 + c, cx + w / 2 - c, cx + w / 2, cx + w / 2,
          cx - w / 2, cx - w / 2]
    ys = [cy + h / 2, cy + h / 2, cy + h / 2 - c, cy - h / 2,
          cy - h / 2, cy + h / 2 - c]
    ax.add_patch(mpatches.Polygon(list(zip(xs, ys)), closed=True,
                                  fc=FC, ec=EC, lw=LW, zorder=3))
    _label(ax, cx, cy, text, fs)


def loop_end(ax, cx, cy, w, h, text, fs=8.5):
    """Граница цикла (конец) — прямоугольник со срезанными нижними углами."""
    c = min(0.32, h * 0.42)
    xs = [cx - w / 2, cx + w / 2, cx + w / 2, cx + w / 2 - c,
          cx - w / 2 + c, cx - w / 2]
    ys = [cy + h / 2, cy + h / 2, cy - h / 2 + c, cy - h / 2,
          cy - h / 2, cy - h / 2 + c]
    ax.add_patch(mpatches.Polygon(list(zip(xs, ys)), closed=True,
                                  fc=FC, ec=EC, lw=LW, zorder=3))
    _label(ax, cx, cy, text, fs)


def comment(ax, x_from, y, x_bracket, text, fs=8, half_h=0.55):
    """Комментарий — квадратная скобка справа с пунктирной привязкой к блоку."""
    ax.plot([x_from, x_bracket], [y, y], ls=(0, (4, 3)), color=EC, lw=1.0,
            zorder=2)
    tick = 0.14
    ax.plot([x_bracket, x_bracket], [y - half_h, y + half_h], color=EC,
            lw=1.1, zorder=3)
    ax.plot([x_bracket, x_bracket + tick], [y + half_h, y + half_h],
            color=EC, lw=1.1, zorder=3)
    ax.plot([x_bracket, x_bracket + tick], [y - half_h, y - half_h],
            color=EC, lw=1.1, zorder=3)
    ax.text(x_bracket + tick + 0.12, y, text, ha='left', va='center',
            fontsize=fs, zorder=4, multialignment='left',
            linespacing=LINESPACING)


def arrow(ax, x1, y1, x2, y2):
    """Стрелка по прямой."""
    ax.annotate('', xy=(x2, y2), xytext=(x1, y1),
                arrowprops=dict(arrowstyle='-|>', color=EC, lw=LW,
                                mutation_scale=12), zorder=2)


def line(ax, pts):
    """Ломаная без наконечника (по списку точек [(x,y),...])."""
    xs = [p[0] for p in pts]
    ys = [p[1] for p in pts]
    ax.plot(xs, ys, color=EC, lw=LW, zorder=2,
            solid_capstyle='round', solid_joinstyle='miter')


def tag(ax, x, y, text, ha='left', fs=8):
    """Метка ветви (Да/Нет)."""
    ax.text(x, y, text, ha=ha, va='center', fontsize=fs, zorder=5)


class Stack:
    """Вертикальная раскладка блоков: считает y-центры по высотам и зазору."""

    def __init__(self, top, gap):
        self.y = top
        self.gap = gap
        self.cy = {}
        self.h = {}

    def add(self, key, h):
        self.cy[key] = self.y - h / 2
        self.h[key] = h
        self.y -= h + self.gap
        return self.cy[key]

    def top(self, key):
        return self.cy[key] + self.h[key] / 2

    def bot(self, key):
        return self.cy[key] - self.h[key] / 2

    def bottom_edge(self):
        return self.y + self.gap
