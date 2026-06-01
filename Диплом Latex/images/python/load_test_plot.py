# -*- coding: utf-8 -*-
"""График нагрузочного тестирования REST API: p95 времени ответа в
зависимости от целевого RPS для трёх основных ручек (открытая модель,
constant-arrival-rate, замер 30 с на точку).

Данные получены прогоном backend/run_load.sh на тестовом стенде
(Apple M1, 16 ГБ, Docker Compose; генератор k6 и серверный стек на одной
машине). Точки, где достигнутый RPS заметно ниже целевого, соответствуют
насыщению системы.

Сохраняет ../load_test.png.
"""
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
from matplotlib.ticker import ScalarFormatter

plt.rcParams.update({
    'font.family': 'DejaVu Sans',
    'font.size': 11,
})

# target RPS -> p95, мс (измеренные значения)
GET = {
    'label': 'GET /api/tasks (чтение)',
    'color': '#1f77b4',
    'marker': 'o',
    'rps':  [50, 100, 200, 400, 800, 1600, 3200, 6400, 12800],
    'p95':  [3.3, 2.5, 2.5, 2.4, 2.1, 6.3, 64.8, 3390.9, 17980.8],
    'sat':  3200,  # последний RPS в пределах бюджета (achieved=target)
}
POST = {
    'label': 'POST /api/tasks (запись)',
    'color': '#2ca02c',
    'marker': 's',
    'rps':  [25, 50, 100, 200, 400, 800, 1600, 3200, 6400],
    'p95':  [4.8, 4.8, 4.2, 3.3, 2.4, 4.7, 22.5, 593.2, 7351.6],
    'sat':  3200,
}
OPT = {
    'label': 'POST /api/routes/optimize (алгоритм + БД)',
    'color': '#d62728',
    'marker': '^',
    'rps':  [5, 10, 20, 40, 80, 160, 320, 640, 1280, 2560, 5120],
    'p95':  [8.6, 10.0, 10.5, 7.8, 5.7, 4.6, 3.6, 6.9, 113.5, 8653.3, 23278.1],
    'sat':  1280,
}

BUDGET_MS = 2000

fig, ax = plt.subplots(figsize=(10, 6.2))

for s in (GET, POST, OPT):
    ax.plot(s['rps'], s['p95'], marker=s['marker'], color=s['color'],
            linewidth=1.8, markersize=6, label=s['label'], zorder=4)
    # вертикальная отметка предела устойчивой работы
    ax.axvline(s['sat'], color=s['color'], linewidth=1.0, linestyle=':',
               alpha=0.55, zorder=2)

ax.axhline(BUDGET_MS, color='#555555', linewidth=1.3, linestyle='--', zorder=3)
ax.text(6, BUDGET_MS * 1.15, 'бюджет p95 = 2000 мс',
        fontsize=9.5, color='#555555', va='bottom')

ax.set_xscale('log')
ax.set_yscale('log')
ax.set_xlabel('Целевой RPS (запросов в секунду)')
ax.set_ylabel('95-й процентиль времени ответа, мс')
ax.set_title('Время ответа основных ручек REST API при росте RPS',
             fontsize=12.5, pad=12)

ax.set_xticks([10, 50, 100, 500, 1000, 5000, 10000])
ax.get_xaxis().set_major_formatter(ScalarFormatter())
ax.set_yticks([1, 10, 100, 1000, 10000])
ax.get_yaxis().set_major_formatter(ScalarFormatter())

ax.grid(True, which='both', linewidth=0.4, alpha=0.5)
ax.legend(loc='upper left', fontsize=9.5, frameon=True,
          edgecolor='#AAAAAA', fancybox=False)

fig.tight_layout()
fig.savefig('../load_test.png', dpi=200, facecolor='white')
print('saved ../load_test.png')
