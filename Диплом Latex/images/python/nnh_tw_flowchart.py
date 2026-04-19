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
W_P  = 4.0    # ширина прямоугольника
W_D  = 4.2    # ширина ромба
W_IO = 3.6    # ширина параллелограмма
W_T  = 2.8    # ширина терминатора
H_T  = 0.55
H_IO = 0.62
GAP  = 0.38

BH = {
    'start':   H_T,
    'input':   0.75,
    'init':    1.25,
    'loop':    0.85,
    'endchk':  1.05,
    'endadd':  0.65,
    'p1_hdr':  1.05,
    'p1_for':  1.05,
    'p1_feas': 1.05,
    'p1_safe': 0.85,
    'p1_look': 1.05,
    'p1_upd':  0.90,
    'con_a':   0.50,
    'p2_chk':  1.05,
    'p2_hdr':  1.05,
    'p2_for':  1.05,
    'p2_upd':  0.78,
    'con_b':   0.50,
    'wait':    0.78,
    'upd':     0.78,
    'append':  0.65,
    'output':  H_IO,
    'end':     H_T,
}

block_order = [
    'start', 'input', 'init', 'loop',
    'endchk', 'endadd',
    'p1_hdr', 'p1_for', 'p1_feas', 'p1_safe', 'p1_look', 'p1_upd', 'con_a',
    'p2_chk', 'p2_hdr', 'p2_for', 'p2_upd', 'con_b',
    'wait', 'upd', 'append', 'output', 'end',
]

yb = {}
y = 30.0
yb['start'] = y
y -= BH['start'] + GAP

for name in block_order[1:]:
    yb[name] = y - BH[name] / 2
    y -= BH[name] + GAP

bottom_y = yb['end'] - BH['end'] / 2
TOTAL_H = 30.0 - bottom_y

hh = {k: v / 2 for k, v in BH.items()}
hh['con_a'] = 0.25
hh['con_b'] = 0.25

fig, ax = plt.subplots(figsize=(10, TOTAL_H / 1.8))
ax.set_xlim(-2.2, 10.5)
ax.set_ylim(bottom_y - 0.5, 30.3)
ax.axis('off')

# --- Блоки ---
terminal(ax, CX, yb['start'],  W_T,  BH['start'],  'НАЧАЛО')
io_block(ax, CX, yb['input'],  W_IO + 0.6, BH['input'],
         'Вход: граф G(V, E), startTimeUnix, constraints')
process (ax, CX, yb['init'],   W_P + 0.2,  BH['init'],
         'cur \u2190 startIdx или узел с ранним окном\nvisited[cur] \u2190 true,  order \u2190 [cur]\n'
         'currentTimeSec \u2190 startTimeUnix + s[cur]\u00d760',
         fs=8)
decision(ax, CX, yb['loop'],   W_D,  BH['loop'],
         '|order| < |V|?')

decision(ax, CX, yb['endchk'], W_D,  BH['endchk'],
         'endIdx задан И все\nпрочие посещены?')
process (ax, CX, yb['endadd'], W_P,  BH['endadd'],
         'Добавить endIdx в маршрут', fs=8)

process (ax, CX, yb['p1_hdr'], W_P + 0.2,  BH['p1_hdr'],
         'ПРОХОД 1: поиск допустимого\nузла + look-ahead\nnext \u2190 \u22121,  bestComp \u2190 \u221e',
         fs=8)
decision(ax, CX, yb['p1_for'], W_D,  BH['p1_for'],
         'Есть непросмотренные\ni \u2209 visited, i \u2260 endIdx?')
decision(ax, CX, yb['p1_feas'], W_D,  BH['p1_feas'],
         'feasible(i, arrival)\nИ prereqsMet(i)?')
process (ax, CX, yb['p1_safe'], W_P,  BH['p1_safe'],
         'safe \u2190 НЕ causesWindowMiss(i, comp)', fs=8)
decision(ax, CX, yb['p1_look'], W_D,  BH['p1_look'],
         'better по (safe, comp,\ndeadline)?')
process (ax, CX, yb['p1_upd'], W_P,  BH['p1_upd'],
         'next \u2190 i; bestComp \u2190 comp\nbestSafe \u2190 safe; bestDeadline \u2190 deadline', fs=8)
connector(ax, CX, yb['con_a'], 0.25, 'A')

decision(ax, CX, yb['p2_chk'], W_D,  BH['p2_chk'],
         'next = \u22121?\n(нет допустимых узлов)')
process (ax, CX, yb['p2_hdr'], W_P + 0.2,  BH['p2_hdr'],
         'ПРОХОД 2 (резервный): выбор\nпо мин. completionTime\nbest \u2190 \u221e',
         fs=8)
decision(ax, CX, yb['p2_for'], W_D,  BH['p2_for'],
         'Есть непросмотренные\ni \u2209 visited, i \u2260 endIdx?')
process (ax, CX, yb['p2_upd'], W_P,  BH['p2_upd'],
         'comp < best? \u2192\nbest \u2190 comp,  next \u2190 i', fs=8)
connector(ax, CX, yb['con_b'], 0.25, '\u0411')

process (ax, CX, yb['wait'],   W_P,  BH['wait'],
         'arrival \u2190 currentTimeSec + travelSec[cur][next]\nwait \u2190 max(0, windowStart[next] \u2212 arrival)',
         fs=8)
process (ax, CX, yb['upd'],    W_P,  BH['upd'],
         'currentTimeSec \u2190 arrival + wait + s[next]\u00d760\nvisited[next] \u2190 true,  cur \u2190 next',
         fs=8)
process (ax, CX, yb['append'], W_P,  BH['append'],
         'order.append(next)', fs=8)

io_block(ax, CX, yb['output'], W_IO + 0.2, BH['output'],
         'Выход: order[],  тайминги[], статистика')
terminal(ax, CX, yb['end'],    W_T,  BH['end'],  'КОНЕЦ')

# --- Стрелки ---
main_seq = [
    ('start',  'input'),
    ('input',  'init'),
    ('init',   'loop'),
    ('endchk', 'endadd'),
    ('p1_hdr', 'p1_for'),
    ('p1_safe','p1_look'),
    ('p1_upd', 'con_a'),
    ('con_a',  'p2_chk'),
    ('p2_hdr', 'p2_for'),
    ('p2_upd', 'con_b'),
    ('con_b',  'wait'),
    ('wait',   'upd'),
    ('upd',    'append'),
    ('output', 'end'),
]
for (ak, bk) in main_seq:
    arr(ax, CX, yb[ak] - hh[ak], CX, yb[bk] + hh[bk])

# Да-ветки
arr(ax, CX, yb['loop']    - hh['loop'],    CX, yb['endchk']  + hh['endchk'],  'Да', side='right')
arr(ax, CX, yb['endchk']  - hh['endchk'],  CX, yb['endadd']  + hh['endadd'],  'Да', side='right')
arr(ax, CX, yb['p1_for']  - hh['p1_for'],  CX, yb['p1_feas'] + hh['p1_feas'], 'Да', side='right')
arr(ax, CX, yb['p1_feas'] - hh['p1_feas'], CX, yb['p1_safe'] + hh['p1_safe'], 'Да', side='right')
arr(ax, CX, yb['p1_look'] - hh['p1_look'], CX, yb['p1_upd']  + hh['p1_upd'],  'Да', side='right')
arr(ax, CX, yb['p2_chk']  - hh['p2_chk'],  CX, yb['p2_hdr']  + hh['p2_hdr'],  'Да', side='right')
arr(ax, CX, yb['p2_for']  - hh['p2_for'],  CX, yb['p2_upd']  + hh['p2_upd'],  'Да', side='right')

# Нет-ветки
LX  = -1.5
RX  =  9.8

# loop → Нет → output
hv(ax, CX + W_D/2, yb['loop'], RX)
vv(ax, RX, yb['loop'], yb['output'])
ax.annotate('', xy=(CX + W_IO/2 + 0.52, yb['output']),
            xytext=(RX, yb['output']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX + W_D/2 + 0.07, yb['loop'] + 0.14, 'Нет', fontsize=8)

# endchk → Нет → p1_hdr
hv(ax, CX - W_D/2, yb['endchk'], LX)
vv(ax, LX, yb['endchk'], yb['p1_hdr'])
ax.annotate('', xy=(CX - W_P/2 - 0.1, yb['p1_hdr']),
            xytext=(LX, yb['p1_hdr']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX - W_D/2 - 0.07, yb['endchk'] + 0.14, 'Нет', ha='right', fontsize=8)

# endadd → output (справа)
RX2 = 9.0
hv(ax, CX + W_P/2, yb['endadd'], RX2)
vv(ax, RX2, yb['endadd'], yb['output'])
ax.annotate('', xy=(CX + W_IO/2 + 0.42, yb['output']),
            xytext=(RX2, yb['output']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)

# p1_for → Нет → con_a
hv(ax, CX - W_D/2, yb['p1_for'], LX)
vv(ax, LX, yb['p1_for'], yb['con_a'])
ax.annotate('', xy=(CX - 0.25, yb['con_a']),
            xytext=(LX, yb['con_a']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX - W_D/2 - 0.07, yb['p1_for'] + 0.14, 'Нет', ha='right', fontsize=8)

# p1_feas → Нет → p1_for (петля)
RX3 = 8.8
hv(ax, CX + W_D/2, yb['p1_feas'], RX3)
vv(ax, RX3, yb['p1_feas'], yb['p1_for'])
ax.annotate('', xy=(CX + W_D/2, yb['p1_for']),
            xytext=(RX3, yb['p1_for']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX + W_D/2 + 0.07, yb['p1_feas'] + 0.14, 'Нет', fontsize=8)

# p1_look → Нет → p1_for (петля)
RX4 = 9.5
hv(ax, CX + W_D/2, yb['p1_look'], RX4)
vv(ax, RX4, yb['p1_look'], yb['p1_for'])
ax.annotate('', xy=(CX + W_D/2 + 0.01, yb['p1_for'] - 0.1),
            xytext=(RX4, yb['p1_for'] - 0.1),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX + W_D/2 + 0.07, yb['p1_look'] + 0.14, 'Нет', fontsize=8)

# p2_chk → Нет → con_b
LX2 = -1.5
hv(ax, CX - W_D/2, yb['p2_chk'], LX2)
vv(ax, LX2, yb['p2_chk'], yb['con_b'])
ax.annotate('', xy=(CX - 0.25, yb['con_b']),
            xytext=(LX2, yb['con_b']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX - W_D/2 - 0.07, yb['p2_chk'] + 0.14, 'Нет', ha='right', fontsize=8)

# p2_for → Нет → con_b
LX3 = -0.5
hv(ax, CX - W_D/2, yb['p2_for'], LX3)
vv(ax, LX3, yb['p2_for'], yb['con_b'])
ax.annotate('', xy=(CX - 0.25, yb['con_b']),
            xytext=(LX3, yb['con_b']),
            arrowprops=dict(arrowstyle='->', color='black',
                            lw=1.2, mutation_scale=11), zorder=2)
ax.text(CX - W_D/2 - 0.07, yb['p2_for'] + 0.14, 'Нет', ha='right', fontsize=8)

# append → возврат к loop
LX4 = -2.0
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
