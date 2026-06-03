"""
Иллюстрация задачи VRPTW (Vehicle Routing Problem with Time Windows).
Три узла — тот же пример, что используется в трассировке алгоритма NNH-TW:
  v0: Начальная точка  [08:00-09:00]  s=30 мин
  v1: Точка А          [10:00-12:00]  s=20 мин
  v2: Точка Б          [09:00-10:00]  s=15 мин
Матрица времён: t01=40, t02=15, t12=20 мин.
Оптимальный маршрут (NNH-TW): v0 → v2 → v1.
"""

import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
import numpy as np

plt.rcParams.update({
    'font.family': 'DejaVu Sans',
    'font.size': 11,
})

fig, ax = plt.subplots(figsize=(10, 7))
ax.set_xlim(0, 10)
ax.set_ylim(0, 8)
ax.axis('off')

# ── Позиции узлов ──────────────────────────────────────────────────────────
pos = {
    0: (5.0, 6.5),   # Начальная точка — вверху по центру
    1: (1.8, 2.2),   # Точка А — внизу слева
    2: (8.2, 2.2),   # Точка Б — внизу справа
}

# ── Рёбра ─────────────────────────────────────────────────────────────────
# (from, to, travel_time_min, is_route)
edges = [
    (0, 2, 15, True),    # v0→v2: маршрут, шаг 1
    (2, 1, 20, True),    # v2→v1: маршрут, шаг 2
    (0, 1, 40, False),   # v0→v1: не выбрано
]

NODE_R = 0.55   # радиус узла

def edge_endpoints(pi, pj, r=NODE_R):
    """Возвращает точки начала и конца стрелки (с отступом r от центра узла)."""
    xi, yi = pi; xj, yj = pj
    dx, dy = xj - xi, yj - yi
    length = (dx**2 + dy**2) ** 0.5
    ux, uy = dx / length, dy / length
    return (xi + ux * r, yi + uy * r), (xj - ux * r, yj - uy * r)

for (i, j, t, is_route) in edges:
    pi, pj = pos[i], pos[j]
    color = '#111111' if is_route else '#BBBBBB'
    lw    = 2.2      if is_route else 0.9
    ls    = '-'      if is_route else '--'
    zorder = 3       if is_route else 1

    # Линия
    ax.plot([pi[0], pj[0]], [pi[1], pj[1]],
            color=color, lw=lw, ls=ls, zorder=zorder, solid_capstyle='round')

    # Стрелка на маршрутных рёбрах
    if is_route:
        s, e = edge_endpoints(pi, pj)
        ax.annotate('', xy=e, xytext=s,
                    arrowprops=dict(arrowstyle='->', color='#111111',
                                    lw=2.2, mutation_scale=16),
                    zorder=4)

    # Метка времени на ребре (чуть выше средины, с белым фоном)
    mx, my = (pi[0] + pj[0]) / 2, (pi[1] + pj[1]) / 2
    dx, dy = pj[0] - pi[0], pj[1] - pi[1]
    length = (dx**2 + dy**2) ** 0.5
    # перпендикулярный сдвиг
    nx, ny = -dy / length, dx / length
    shift = 0.38 if is_route else 0.30
    tx, ty = mx + nx * shift, my + ny * shift
    ax.text(tx, ty, f'{t} мин',
            ha='center', va='center', fontsize=9,
            color='#111111' if is_route else '#888888',
            bbox=dict(boxstyle='round,pad=0.15', fc='white',
                      ec='none', alpha=0.9),
            zorder=5)

# ── Узлы ──────────────────────────────────────────────────────────────────
def draw_node(ax, x, y, label, desc_lines, is_start=False):
    """Рисует узел и текстовый блок вплотную к нему."""
    if is_start:
        # Ромб для начальной точки
        d = NODE_R * 1.05
        diamond = plt.Polygon(
            [[x, y + d], [x + d, y], [x, y - d], [x - d, y]],
            closed=True, fc='#222222', ec='#111111', lw=1.5, zorder=5)
        ax.add_patch(diamond)
        ax.text(x, y, label, ha='center', va='center',
                fontsize=12, fontweight='bold', color='white', zorder=6)
    else:
        circle = plt.Circle((x, y), NODE_R,
                             fc='white', ec='#222222', lw=1.8, zorder=5)
        ax.add_patch(circle)
        ax.text(x, y, label, ha='center', va='center',
                fontsize=12, fontweight='bold', color='#111111', zorder=6)

    # Описание вплотную к узлу
    desc_text = '\n'.join(desc_lines)
    # Вычислить позицию блока описания (всегда снаружи относительно центра графа)
    return desc_text

# Узел v0 — описание снизу (между v1 и v2)
draw_node(ax, *pos[0], r'$v_0$',
          ['Начальная точка', '[08:00–09:00]', r'$s_0$ = 30 мин'], is_start=True)
ax.text(pos[0][0], pos[0][1] - NODE_R - 0.15,
        'Начальная точка\n[08:00–09:00]  $s_0$ = 30 мин',
        ha='center', va='top', fontsize=9, color='#222222',
        linespacing=1.45, zorder=6,
        bbox=dict(boxstyle='round,pad=0.25', fc='#F5F5F5',
                  ec='#AAAAAA', lw=0.8))

# Узел v1 — описание слева
draw_node(ax, *pos[1], r'$v_1$', [], is_start=False)
ax.text(pos[1][0] - NODE_R - 0.15, pos[1][1],
        'Точка А\n[10:00–12:00]\n$s_1$ = 20 мин',
        ha='right', va='center', fontsize=9, color='#222222',
        linespacing=1.45, zorder=6,
        bbox=dict(boxstyle='round,pad=0.25', fc='#F5F5F5',
                  ec='#AAAAAA', lw=0.8))

# Узел v2 — описание справа
draw_node(ax, *pos[2], r'$v_2$', [], is_start=False)
ax.text(pos[2][0] + NODE_R + 0.15, pos[2][1],
        'Точка Б\n[09:00–10:00]\n$s_2$ = 15 мин',
        ha='left', va='center', fontsize=9, color='#222222',
        linespacing=1.45, zorder=6,
        bbox=dict(boxstyle='round,pad=0.25', fc='#F5F5F5',
                  ec='#AAAAAA', lw=0.8))

# ── Порядковые номера маршрута ─────────────────────────────────────────────
order_map = {0: ('1', 0.75, 0.55),
             2: ('2', 0.75, 0.55),
             1: ('3', 0.75, 0.55)}
badge_offsets = {
    0: (+0.75, +0.60),
    2: (+0.75, +0.60),
    1: (+0.75, +0.60),
}
for idx, (num, _, __) in order_map.items():
    x, y = pos[idx]
    ox, oy = badge_offsets[idx]
    ax.text(x + ox, y + oy, num,
            ha='center', va='center', fontsize=9, color='white',
            fontweight='bold', zorder=7,
            bbox=dict(boxstyle='circle,pad=0.22', fc='#333333', ec='none'))

# ── Легенда ────────────────────────────────────────────────────────────────
legend_items = [
    mpatches.Patch(fc='#222222', ec='#111111',
                   label='Начальная точка $v_0$'),
    mpatches.Patch(fc='white', ec='#222222',
                   label=r'Задача $v_i$:  [окно]  $s_i$ — длительность'),
    plt.Line2D([0], [0], color='#111111', lw=2.2,
               label='Оптимальный маршрут (NNH-TW)'),
    plt.Line2D([0], [0], color='#BBBBBB', lw=0.9, ls='--',
               label='Возможное ребро (не выбрано)'),
    mpatches.Patch(fc='#333333', ec='none',
                   label='Номер в маршруте'),
]
ax.legend(handles=legend_items, loc='lower center',
          bbox_to_anchor=(0.5, -0.03), ncol=3,
          fontsize=8.5, frameon=True, edgecolor='#AAAAAA', fancybox=False)

# ── Заголовок ─────────────────────────────────────────────────────────────
plt.tight_layout()
plt.savefig(
    '/Users/semyontyutichkin/Documents/Политех лабы/Диплом/diploma/'
    'Диплом Latex/images/vrptw_illustration.png',
    dpi=200, bbox_inches='tight', facecolor='white')
print("vrptw_illustration.png saved")
