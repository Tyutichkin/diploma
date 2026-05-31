# -*- coding: utf-8 -*-
"""Подробная блок-схема алгоритма NNH-TW (ГОСТ 19.701-90), две колонки.

Стиль (как в учебных примерах bubbleSort/insertSort):
- минимум текста: каждый блок — одна операция в синтаксисе (`x := ...`);
- условия — выражения внутри ромба;
- циклы — символами «граница цикла» (срезанные углы) с заголовком
  `пока …` / `для …`; обратная дуга не рисуется (подразумевается парой
  начало/конец цикла);
- сигнатура процедуры — комментарием на скобке справа от «начало»;
- связь колонок — соединителями-окружностями (А, Б).

Левая колонка: основной цикл Ц1, проверка endIdx, Проход 1 (цикл Ц2).
Правая колонка: Проход 2 (цикл Ц3), расчёт времени, конец цикла, вывод.
Размеры фигур подбираются под габарит текста (text_size) — текст не
выходит за границы.
"""
import matplotlib.pyplot as plt
from gost_shapes import (terminal, process, decision, io_block, loop_start,
                         loop_end, connector, comment, arrow, line, tag,
                         text_size, Stack)

DPI = 100
FS = 8.5
PADX, PADY = 0.36, 0.24
DIA_KX, DIA_KY = 0.42, 0.44
GAP = 0.42
CR = 0.34          # радиус соединителя

DEF = {
    'start':  ('terminal',   'начало', 9),
    'prep':   ('process',    'prereqs := предш(C);\nendIdx := C.endIdx', FS),
    'init':   ('process',    'cur := старт(G, C);  visited[cur] := истина\norder := [cur];  t := t0 + service[cur]', FS),
    'l1s':    ('loop_start',  'Ц1: пока |order| < |V|', FS),
    'endchk': ('decision',    'endIdx ≥ 0 и\nостался лишь endIdx?', FS),
    'p1init': ('process',     'next := −1;  best := ∞;\nbestSafe := ложь', FS),
    'l2s':    ('loop_start',  'Ц2: для i ∉ visited, i ≠ endIdx', FS),
    'feas':   ('decision',    'prereq(i) и\narrival(i) ≤ winEnd(i)?', FS),
    'p1eval': ('process',     'comp := completion(i);\nsafe := не блокирует окна(i)', FS),
    'better': ('decision',    '(safe, comp, dl) лучше\n(bestSafe, best, bestDl)?', FS),
    'p1set':  ('process',     'next := i;  best := comp;\nbestSafe := safe;  bestDl := dl', FS),
    'l2e':    ('loop_end',    'Ц2', FS),
    'connA_o': ('conn',       'А', 9),
    'addr':   ('process',     'order += endIdx', FS),
    'connB_o': ('conn',       'Б', 9),
    # правая колонка
    'connA_i': ('conn',       'А', 9),
    'found':  ('decision',    'next = −1?', FS),
    'l3s':    ('loop_start',  'Ц3: для i ∉ visited, i ≠ endIdx', FS),
    'cmp':    ('decision',    'completion(i) < best?', FS),
    'p2set':  ('process',     'next := i;  best := completion(i)', FS),
    'l3e':    ('loop_end',    'Ц3', FS),
    'arr':    ('process',     'arrival := t + travel[cur][next];\nwait := max(0, winStart[next] − arrival)', FS),
    'upd':    ('process',     't := arrival + wait + service[next];\nvisited[next] := истина;\norder += next;  cur := next', FS),
    'l1e':    ('loop_end',    'Ц1', FS),
    'connB_i': ('conn',       'Б', 9),
    'output': ('io',          'вывод: order, тайминги, статистика', FS),
    'end':    ('terminal',    'конец', 9),
}

W, H, KIND = {}, {}, {}
for k, (kind, text, fs) in DEF.items():
    KIND[k] = kind
    if kind == 'conn':
        W[k] = H[k] = 2 * CR
        continue
    tw, th = text_size(text, fs, DPI)
    if kind == 'terminal':
        h = th + 2 * PADY; w = tw + 2 * PADX + h
    elif kind == 'process':
        w, h = tw + 2 * PADX, th + 2 * PADY
    elif kind in ('loop_start', 'loop_end'):
        w, h = tw + 2 * PADX + 0.4, th + 2 * PADY
    elif kind == 'io':
        h = th + 2 * PADY; w = tw + 2 * PADX + 0.84 * h
    elif kind == 'decision':
        w, h = tw / DIA_KX, th / DIA_KY
    W[k], H[k] = w, h

LEFT = ['start', 'prep', 'init', 'l1s', 'endchk', 'p1init', 'l2s', 'feas',
        'p1eval', 'better', 'p1set', 'l2e', 'connA_o']
RIGHT = ['connA_i', 'found', 'l3s', 'cmp', 'p2set', 'l3e', 'arr', 'upd',
         'l1e', 'output', 'end']
EXTRA_L = {'l2e': 0.5, 'connA_o': 0.3}
EXTRA_R = {'l3e': 0.5, 'arr': 0.5, 'l1e': 0.5}

sL = Stack(top=0.0, gap=GAP)
for k in LEFT:
    sL.y -= EXTRA_L.get(k, 0.0); sL.add(k, H[k])
sR = Stack(top=0.0, gap=GAP)
for k in RIGHT:
    sR.y -= EXTRA_R.get(k, 0.0); sR.add(k, H[k])

CY = {}
CY.update(sL.cy); CY.update(sR.cy)
CY['addr'] = CY['endchk']
CY['connB_o'] = CY['endchk'] - H['endchk'] / 2 - 0.7

LX = 0.0
# Геометрия левой колонки и боковых элементов
R2 = max(W['feas'], W['better'], W['p1set']) / 2 + 1.0      # continue Ц2
ARL = W['endchk'] / 2 + 1.0
ARX = ARL + W['addr'] / 2
left_right = max(R2, ARX + W['addr'] / 2, ARX + CR)
maxRW = max(W[k] for k in RIGHT if KIND[k] != 'conn')
RX = left_right + 1.7 + maxRW / 2
CY_bottom = min(sL.bottom_edge(), sR.bottom_edge())

# Рельсы правой колонки
R3 = RX + max(W['cmp'], W['p2set']) / 2 + 0.9               # continue Ц3
RFOUND = RX + maxRW / 2 + 1.7                                # found «нет» → arr
CONNB_IX = RX - W['l1e'] / 2 - 1.0
CY['connB_i'] = CY['l1e'] + H['l1e'] / 2 + 0.78

XPOS = {k: LX for k in LEFT}
XPOS.update({k: RX for k in RIGHT})
XPOS['addr'] = ARX
XPOS['connB_o'] = ARX
XPOS['connB_i'] = CONNB_IX

xmin = -max(W[k] for k in LEFT if KIND[k] != 'conn') / 2 - 0.4
xmax = RFOUND + 0.5
ymax = 0.5
ymin = CY_bottom - 0.5
M = 0.3
fig = plt.figure(figsize=(xmax - xmin + 2 * M, ymax - ymin + 2 * M), dpi=DPI)
ax = fig.add_axes([0, 0, 1, 1])
ax.set_xlim(xmin - M, xmax + M)
ax.set_ylim(ymin - M, ymax + M)
ax.set_aspect('equal')
ax.axis('off')


def draw(k):
    kind = KIND[k]
    x, y = XPOS[k], CY[k]
    if kind == 'conn':
        connector(ax, x, y, CR, DEF[k][1], fs=DEF[k][2]); return
    {'terminal': terminal, 'process': process, 'decision': decision,
     'io': io_block, 'loop_start': loop_start, 'loop_end': loop_end}[kind](
        ax, x, y, W[k], H[k], DEF[k][1], fs=DEF[k][2])


for k in DEF:
    draw(k)

# Комментарий-сигнатура
comment(ax, W['start'] / 2, CY['start'], 1.5,
        'NNH-TW(G, t0, C)\n// C = {startIdx, endIdx,\n//      предшествования}',
        fs=8, half_h=0.6)


def top(k):
    return CY[k] + H[k] / 2


def bot(k):
    return CY[k] - H[k] / 2


def vlink(a, b):
    arrow(ax, XPOS[a], bot(a), XPOS[b], top(b))


# Спайн левой колонки
for a, b in [('start', 'prep'), ('prep', 'init'), ('init', 'l1s'),
             ('l1s', 'endchk'), ('endchk', 'p1init'), ('p1init', 'l2s'),
             ('l2s', 'feas'), ('feas', 'p1eval'), ('p1eval', 'better'),
             ('better', 'p1set'), ('p1set', 'l2e'), ('l2e', 'connA_o')]:
    vlink(a, b)
# Спайн правой колонки
for a, b in [('connA_i', 'found'), ('found', 'l3s'), ('l3s', 'cmp'),
             ('cmp', 'p2set'), ('p2set', 'l3e'), ('l3e', 'arr'),
             ('arr', 'upd'), ('upd', 'l1e'), ('l1e', 'output'),
             ('output', 'end')]:
    vlink(a, b)

for k, lbl, col in [('endchk', 'нет', LX), ('feas', 'да', LX),
                    ('better', 'да', LX), ('found', 'да', RX),
                    ('cmp', 'да', RX)]:
    tag(ax, col + 0.12, bot(k) - 0.16, lbl)


def skip(src, target, rail_x, label='нет', gap=0.30):
    yj = top(target) + gap
    line(ax, [(XPOS[src] + W[src] / 2, CY[src]), (rail_x, CY[src]),
              (rail_x, yj), (XPOS[target], yj)])
    tag(ax, XPOS[src] + W[src] / 2 + 0.1, CY[src] + 0.15, label)

# Левая колонка: continue Ц2
skip('feas', 'l2e', R2)
skip('better', 'l2e', R2)
# Правая колонка: continue Ц3 и обход Ц3
skip('cmp', 'l3e', R3)
skip('found', 'arr', RFOUND)

# endchk «да» → addRoute → соединитель Б
line(ax, [(XPOS['endchk'] + W['endchk'] / 2, CY['endchk']), (ARL, CY['endchk'])])
arrow(ax, ARL - 0.001, CY['endchk'], ARL, CY['endchk'])
tag(ax, XPOS['endchk'] + W['endchk'] / 2 + 0.1, CY['endchk'] + 0.15, 'да')
arrow(ax, ARX, bot('addr'), ARX, CY['connB_o'] + CR)

# Соединитель Б (вход) → конец цикла Ц1 (слияние со спайном upd→Ц1)
yj = top('l1e') + 0.30
line(ax, [(CONNB_IX, CY['connB_i'] - CR), (CONNB_IX, yj), (XPOS['l1e'], yj)])

fig.savefig('../nnh_tw_flowchart.png', dpi=200, facecolor='white')
print('nnh_tw_flowchart.png saved')
