import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

plt.rcParams.update({'font.family': 'DejaVu Sans', 'font.size': 9})

def terminal(ax, cx, cy, w=2.8, h=0.55, text='', fs=9):
    ell = mpatches.Ellipse((cx, cy), w, h,
                           fc='white', ec='black', lw=1.3, zorder=3)
    ax.add_patch(ell)
    ax.text(cx, cy, text, ha='center', va='center',
            fontsize=fs, fontweight='bold', zorder=4)

def process(ax, cx, cy, w=3.6, h=0.65, text='', fs=8.5):
    rect = plt.Rectangle((cx - w/2, cy - h/2), w, h,
                          fc='white', ec='black', lw=1.2, zorder=3)
    ax.add_patch(rect)
    ax.text(cx, cy, text, ha='center', va='center',
            fontsize=fs, zorder=4, multialignment='center', linespacing=1.35)

def decision(ax, cx, cy, w=3.8, h=0.9, text='', fs=8.5):
    xs = [cx, cx+w/2, cx, cx-w/2, cx]
    ys = [cy+h/2, cy, cy-h/2, cy, cy+h/2]
    poly = mpatches.Polygon(list(zip(xs, ys)), closed=True,
                             fc='white', ec='black', lw=1.2, zorder=3)
    ax.add_patch(poly)
    ax.text(cx, cy, text, ha='center', va='center',
            fontsize=fs, zorder=4, multialignment='center', linespacing=1.3)

def io_block(ax, cx, cy, w=3.4, h=0.60, text='', fs=8.5):
    d = 0.22
    xs = [cx-w/2+d, cx+w/2+d, cx+w/2-d, cx-w/2-d]
    ys = [cy+h/2,   cy+h/2,   cy-h/2,   cy-h/2]
    poly = mpatches.Polygon(list(zip(xs, ys)), closed=True,
                             fc='white', ec='black', lw=1.2, zorder=3)
    ax.add_patch(poly)
    ax.text(cx, cy, text, ha='center', va='center',
            fontsize=fs, zorder=4, multialignment='center')

def connector(ax, cx, cy, r=0.25, text='', fs=8):
    c = plt.Circle((cx, cy), r, fc='white', ec='black', lw=1.2, zorder=3)
    ax.add_patch(c)
    ax.text(cx, cy, text, ha='center', va='center', fontsize=fs,
            fontweight='bold', zorder=4)

def arr(ax, x1, y1, x2, y2, label='', side='right', fs=8):
    ax.annotate('', xy=(x2, y2), xytext=(x1, y1),
                arrowprops=dict(arrowstyle='->', color='black',
                                lw=1.2, mutation_scale=11), zorder=2)
    if label:
        mx, my = (x1+x2)/2, (y1+y2)/2
        off = 0.13 if side == 'right' else -0.13
        ha  = 'left' if side == 'right' else 'right'
        ax.text(mx+off, my, label, ha=ha, va='center', fontsize=fs)

def hv(ax, x1, y, x2, color='black', lw=1.2):
    ax.plot([x1, x2], [y, y], color=color, lw=lw, zorder=2)

def vv(ax, x, y1, y2, color='black', lw=1.2):
    ax.plot([x, x], [y1, y2], color=color, lw=lw, zorder=2)

CX   = 4.0
W_P  = 3.8    # ширина прямоугольника
W_D  = 4.0    # ширина ромба
W_IO = 3.4    # ширина параллелограмма
W_T  = 2.8    # ширина терминатора
H_T  = 0.55   # высота терминатора
H_IO = 0.62   # высота параллелограмма
GAP  = 0.38   # вертикальный зазор между блоками

# Высота каждого блока (подобрана под кол-во строк текста)
BH = {
    'start':  H_T,
    'input':  H_IO,
    'init':   1.05,   # 3 строки процесса
    'loop':   0.85,   # 1 строка, ромб
    'p1_hdr': 1.05,   # 3 строки процесса
    'p1_for': 1.05,   # 2 строки, ромб
    'p1_cnd': 1.05,   # 2 строки, ромб
    'p1_upd': 0.78,   # 2 строки процесса
    'con_a':  0.50,   # диаметр соединителя
    'p2_chk': 1.05,   # 2 строки, ромб
    'p2_hdr': 1.05,   # 3 строки процесса
    'p2_for': 1.05,   # 2 строки, ромб
    'p2_upd': 0.78,   # 2 строки процесса
    'con_b':  0.50,   # диаметр соединителя
    'wait':   0.78,   # 2 строки процесса
    'upd':    0.78,   # 2 строки процесса
    'append': 0.65,   # 1 строка процесса
    'output': H_IO,
    'end':    H_T,
}

# Порядок блоков сверху вниз
block_order = [
    'start', 'input', 'init', 'loop',
    'p1_hdr', 'p1_for', 'p1_cnd', 'p1_upd', 'con_a',
    'p2_chk', 'p2_hdr', 'p2_for', 'p2_upd', 'con_b',
    'wait', 'upd', 'append', 'output', 'end',
]

yb = {}
y = 25.0

# 'start': центр в точке y=25.0; далее y смещается на полную высоту + GAP
# (такая же логика, как в оригинале)
yb['start'] = y
y -= BH['start'] + GAP

for name in block_order[1:]:
    yb[name] = y - BH[name] / 2
    y -= BH[name] + GAP

# Нижняя граница последнего блока (КОНЕЦ)
bottom_y = yb['end'] - BH['end'] / 2
TOTAL_H   = 25.0 - bottom_y

# Словарь полуразмеров для удобного подключения стрелок
hh = {k: v / 2 for k, v in BH.items()}
# Для соединителей полуразмер = радиус
hh['con_a'] = 0.25
hh['con_b'] = 0.25

fig, ax = plt.subplots(figsize=(9, TOTAL_H / 1.9))
ax.set_xlim(-1.8, 9.8)
ax.set_ylim(bottom_y - 0.5, 25.3)   # запас снизу, чтобы КОНЕЦ не обрезался
ax.axis('off')

# ─── Блоки ────────────────────────────────────────────────────────────────
terminal(ax, CX, yb['start'],  W_T,  BH['start'],  'НАЧАЛО')
io_block(ax, CX, yb['input'],  W_IO, BH['input'],
         'Вход: граф G(V, E), startTimeMins')
process (ax, CX, yb['init'],   W_P,  BH['init'],
         'cur \u2190 узел с наиболее ранним окном\nvisited[cur] \u2190 true,  order \u2190 [cur]\ncurrentTime \u2190 startTimeMins + s[cur]',
         fs=8)
decision(ax, CX, yb['loop'],   W_D,  BH['loop'],
         '|order| < |V|?')

process (ax, CX, yb['p1_hdr'], W_P,  BH['p1_hdr'],
         '\u041f\u0420\u041e\u0425\u041e\u0414 1: поиск ближайшего\nдопустимого узла\nnext \u2190 \u22121,  best \u2190 \u221e',
         fs=8)
decision(ax, CX, yb['p1_for'], W_D,  BH['p1_for'],
         'Есть непросмотренные\nузлы i \u2209 visited?')
decision(ax, CX, yb['p1_cnd'], W_D,  BH['p1_cnd'],
         'feasible(i, arrival)\n\u0418 t[cur][i] < best?')
process (ax, CX, yb['p1_upd'], W_P,  BH['p1_upd'],
         'best \u2190 t[cur][i]\nnext \u2190 i', fs=8)
connector(ax, CX, yb['con_a'], 0.25, 'A')

decision(ax, CX, yb['p2_chk'], W_D,  BH['p2_chk'],
         'next = \u22121?\n(нет допустимых узлов)')
process (ax, CX, yb['p2_hdr'], W_P,  BH['p2_hdr'],
         '\u041f\u0420\u041e\u0425\u041e\u0414 2 (резервный): выбор\nближайшего без учёта окна\nbest \u2190 \u221e',
         fs=8)
decision(ax, CX, yb['p2_for'], W_D,  BH['p2_for'],
         'Есть непросмотренные\nузлы i \u2209 visited?')
process (ax, CX, yb['p2_upd'], W_P,  BH['p2_upd'],
         't[cur][i] < best? \u2192\nbest \u2190 t[cur][i],  next \u2190 i', fs=8)
connector(ax, CX, yb['con_b'], 0.25, '\u0411')

process (ax, CX, yb['wait'],   W_P,  BH['wait'],
         'arrival \u2190 currentTime + t[cur][next] / 60\nwait \u2190 max(0, a[next] \u2212 arrival)',
         fs=8)
process (ax, CX, yb['upd'],    W_P,  BH['upd'],
         'currentTime \u2190 arrival + wait + s[next]\nvisited[next] \u2190 true,  cur \u2190 next',
         fs=8)
process (ax, CX, yb['append'], W_P,  BH['append'],
         'order.append(next)', fs=8)

io_block(ax, CX, yb['output'], W_IO, BH['output'],
         'Выход: order[],  статистика маршрута')
terminal(ax, CX, yb['end'],    W_T,  BH['end'],  '\u041a\u041e\u041d\u0415\u0426')

# ─── Стрелки главной оси (вертикальные переходы между блоками) ────────────
main_seq = [
    ('start',  'input'),
    ('input',  'init'),
    ('init',   'loop'),
    # loop →Да→ p1_hdr — отдельно (с меткой)
    ('p1_hdr', 'p1_for'),
    # p1_for →Да→ p1_cnd — отдельно
    # p1_cnd →Да→ p1_upd — отдельно
    ('p1_upd', 'con_a'),
    ('con_a',  'p2_chk'),
    # p2_chk →Да→ p2_hdr — отдельно
    ('p2_hdr', 'p2_for'),
    # p2_for →Да→ p2_upd — отдельно
    ('p2_upd', 'con_b'),
    ('con_b',  'wait'),
    ('wait',   'upd'),
    ('upd',    'append'),
    # append → loop — петля возврата, отдельно
    ('output', 'end'),
]
for (ak, bk) in main_seq:
    arr(ax, CX, yb[ak] - hh[ak], CX, yb[bk] + hh[bk])

# ─── Ветки «Да» ───────────────────────────────────────────────────────────
arr(ax, CX, yb['loop']   - hh['loop'],   CX, yb['p1_hdr'] + hh['p1_hdr'], '\u0414\u0430', side='right')
arr(ax, CX, yb['p1_for'] - hh['p1_for'], CX, yb['p1_cnd'] + hh['p1_cnd'], '\u0414\u0430', side='right')
arr(ax, CX, yb['p1_cnd'] - hh['p1_cnd'], CX, yb['p1_upd'] + hh['p1_upd'], '\u0414\u0430', side='right')
arr(ax, CX, yb['p2_chk'] - hh['p2_chk'], CX, yb['p2_hdr'] + hh['p2_hdr'], '\u0414\u0430', side='right')
arr(ax, CX, yb['p2_for'] - hh['p2_for'], CX, yb['p2_upd'] + hh['p2_upd'], '\u0414\u0430', side='right')

# ─── Ветки «Нет» ──────────────────────────────────────────────────────────
LX  = -1.3   # левый обходной канал (дальний)
RX  =  9.3   # правый обходной канал

# loop → Нет → output (справа, выход из главного цикла)
hv(ax, CX + W_D/2, yb['loop'], RX)
vv(ax, RX, yb['loop'], yb['output'])
ax.annotate('', xy=(CX + W_IO/2 + 0.22, yb['output']),
            xytext=(RX, yb['output']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX + W_D/2 + 0.07, yb['loop'] + 0.14, '\u041d\u0435\u0442', fontsize=8)

# p1_for → Нет → con_a (слева)
hv(ax, CX - W_D/2, yb['p1_for'], LX)
vv(ax, LX, yb['p1_for'], yb['con_a'])
ax.annotate('', xy=(CX - 0.25, yb['con_a']),
            xytext=(LX, yb['con_a']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX - W_D/2 - 0.07, yb['p1_for'] + 0.14, '\u041d\u0435\u0442', ha='right', fontsize=8)

# p1_cnd → Нет → p1_for (справа, короткая петля возврата к перебору)
RX2 = 8.6
hv(ax, CX + W_D/2, yb['p1_cnd'], RX2)
vv(ax, RX2, yb['p1_cnd'], yb['p1_for'])
ax.annotate('', xy=(CX + W_D/2, yb['p1_for']),
            xytext=(RX2, yb['p1_for']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX + W_D/2 + 0.07, yb['p1_cnd'] + 0.14, '\u041d\u0435\u0442', fontsize=8)

# p2_chk → Нет → con_b (слева, пропуск прохода 2)
LX2 = -1.3
hv(ax, CX - W_D/2, yb['p2_chk'], LX2)
vv(ax, LX2, yb['p2_chk'], yb['con_b'])
ax.annotate('', xy=(CX - 0.25, yb['con_b']),
            xytext=(LX2, yb['con_b']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX - W_D/2 - 0.07, yb['p2_chk'] + 0.14, '\u041d\u0435\u0442', ha='right', fontsize=8)

# p2_for → Нет → con_b (слева, внутренняя петля)
LX3 = -0.5
hv(ax, CX - W_D/2, yb['p2_for'], LX3)
vv(ax, LX3, yb['p2_for'], yb['con_b'])
ax.annotate('', xy=(CX - 0.25, yb['con_b']),
            xytext=(LX3, yb['con_b']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX - W_D/2 - 0.07, yb['p2_for'] + 0.14, '\u041d\u0435\u0442', ha='right', fontsize=8)

# append → возврат к loop (слева, длинная петля)
LX4 = -1.7
hv(ax, CX - W_P/2, yb['append'], LX4)
vv(ax, LX4, yb['append'], yb['loop'])
ax.annotate('', xy=(CX - W_D/2, yb['loop']),
            xytext=(LX4, yb['loop']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)

plt.tight_layout(pad=0.4)
plt.savefig(
    '/Users/semyontyutichkin/Documents/\u041f\u043e\u043b\u0438\u0442\u0435\u0445 \u043b\u0430\u0431\u044b/\u0414\u0438\u043f\u043b\u043e\u043c/diploma/'
    '\u0414\u0438\u043f\u043b\u043e\u043c Latex/images/nnh_tw_flowchart.png',
    dpi=200, bbox_inches='tight', facecolor='white')
print("nnh_tw_flowchart.png saved")
