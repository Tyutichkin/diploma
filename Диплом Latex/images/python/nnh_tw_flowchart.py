# -*- coding: utf-8 -*-
"""Блок-схема алгоритма NNH-TW (ГОСТ 19.701-90), два рисунка.

Схема разбита на два читаемых вертикальных рисунка:
- nnh_tw_main.png   — основной цикл построения маршрута; выбор кандидата
                      вынесен в символ «предопределённый процесс»;
- nnh_tw_select.png — детализация подпрограммы «Выбрать следующую задачу»
                      (проход 1 с просмотром вперёд и резервный проход 2).

Принципы оформления:
- внутри блоков — смысловые формулировки, без псевдокода (полный псевдокод
  приведён отдельным листингом в тексте);
- циклы — обычные ромбы с явной обратной дугой (стрелкой возврата);
- ветви условий подписаны «да/нет» рядом с выходящей линией;
- сигнатура процедуры вынесена в подпись к рисунку, не на схему;
- пояснения терминов приведены в тексте, а не на схеме.
"""
import matplotlib.pyplot as plt
from gost_shapes import (terminal, process, predefined, decision, io_block,
                         arrow, line, tag, text_size)

DPI = 100
FS = 8.5
PADX, PADY = 0.36, 0.26
DIA_KX, DIA_KY = 0.50, 0.52
GAP = 0.55
EC = 'black'


def size_of(kind, text, fs):
    tw, th = text_size(text, fs, DPI)
    if kind == 'terminal':
        h = th + 2 * PADY
        return tw + 2 * PADX + h, h
    if kind == 'process':
        return tw + 2 * PADX, th + 2 * PADY
    if kind == 'predefined':
        return tw + 2 * PADX + 0.5, th + 2 * PADY
    if kind == 'io':
        h = th + 2 * PADY
        return tw + 2 * PADX + 0.84 * h, h
    if kind == 'decision':
        return tw / DIA_KX, th / DIA_KY
    raise ValueError(kind)


DRAW = {'terminal': terminal, 'process': process, 'predefined': predefined,
        'decision': decision, 'io': io_block}


def measure(D):
    W, H, KIND = {}, {}, {}
    for k, (kind, text, fs) in D.items():
        KIND[k] = kind
        W[k], H[k] = size_of(kind, text, fs)
    return W, H, KIND


# ----------------------------------------------------------------------------
#  Рисунок 1 — основной цикл построения маршрута
# ----------------------------------------------------------------------------
def build_main():
    D = {
        'start':  ('terminal',   'начало', 9),
        'prep':   ('process',    'Прочитать ограничения:\nстартовая, финишная задачи\n'
                                 'и порядок выполнения', FS),
        'init':   ('process',    'Инициализировать маршрут\n'
                                 'стартовой задачей\nи текущее время', FS),
        'l1':     ('decision',   'Все задачи\nдобавлены в маршрут?', FS),
        'endq':   ('decision',   'Финишная задача задана\nи осталась последней?', FS),
        'sub':    ('predefined', 'Выбрать следующую\nзадачу (проходы 1-2)', FS),
        'upd':    ('process',    'Вычислить прибытие\nи ожидание; обновить\n'
                                 'время и маршрут', FS),
        'addend': ('process',    'Добавить финишную\nзадачу в конец', FS),
        'output': ('io',         'Вывод: порядок,\nтайминги, статистика', FS),
        'end':    ('terminal',   'конец', 9),
    }
    W, H, KIND = measure(D)
    spine = ['start', 'prep', 'init', 'l1', 'endq', 'sub', 'upd']
    CY, X = {}, {k: 0.0 for k in D}
    y = 0.0
    for k in spine:
        CY[k] = y - H[k] / 2
        y -= H[k] + GAP

    spine_hw = max(W[k] for k in spine) / 2
    # правый столбец: вывод (на уровне l1) и конец
    RX = spine_hw + 2.4 + W['output'] / 2
    X['output'] = RX
    CY['output'] = CY['l1']
    X['end'] = RX
    CY['end'] = CY['output'] - H['output'] / 2 - GAP - H['end'] / 2
    # левый блок «добавить завершающую задачу» на уровне endq
    LXa = -(max(W['endq'], W['sub'], W['upd']) / 2 + 1.9 + W['addend'] / 2)
    X['addend'] = LXa
    CY['addend'] = CY['endq']
    railx = LXa - W['addend'] / 2 - 0.7

    def top(k):
        return CY[k] + H[k] / 2

    def bot(k):
        return CY[k] - H[k] / 2

    yb = bot('upd') - 0.55
    base_ymin = min(bot('end'), yb)

    xmin = railx - 0.5
    xmax = RX + W['output'] / 2 + 0.5
    fig = plt.figure(dpi=DPI)
    ax = fig.add_axes([0, 0, 1, 1])
    ax.set_aspect('equal')
    ax.axis('off')

    for k in D:
        DRAW[KIND[k]](ax, X[k], CY[k], W[k], H[k], D[k][1], fs=D[k][2])

    ymin = base_ymin - 0.4
    ymax = 0.4
    ax.set_xlim(xmin, xmax)
    ax.set_ylim(ymin, ymax)

    # спайн
    for a, b in [('start', 'prep'), ('prep', 'init'), ('init', 'l1')]:
        arrow(ax, X[a], bot(a), X[b], top(b))
    # l1 «нет» -> endq, endq «нет» -> sub, sub -> upd
    arrow(ax, 0, bot('l1'), 0, top('endq'))
    arrow(ax, 0, bot('endq'), 0, top('sub'))
    arrow(ax, 0, bot('sub'), 0, top('upd'))
    tag(ax, 0.12, bot('l1') - 0.18, 'нет')
    tag(ax, 0.12, bot('endq') - 0.18, 'нет')

    # l1 «да» -> вывод -> конец
    arrow(ax, W['l1'] / 2, CY['l1'], RX - W['output'] / 2, CY['l1'])
    tag(ax, W['l1'] / 2 + 0.12, CY['l1'] + 0.18, 'да')
    arrow(ax, RX, bot('output'), RX, top('end'))

    # endq «да» -> добавить завершающую задачу
    arrow(ax, -W['endq'] / 2, CY['endq'], LXa + W['addend'] / 2, CY['endq'])
    tag(ax, -W['endq'] / 2 - 0.12, CY['endq'] + 0.18, 'да', ha='right')

    # обратная дуга к l1: upd и addend -> рельс -> вверх -> в l1
    line(ax, [(0, bot('upd')), (0, yb), (railx, yb)])
    line(ax, [(LXa - W['addend'] / 2, CY['endq']), (railx, CY['endq'])])
    line(ax, [(railx, yb), (railx, CY['l1'])])
    arrow(ax, railx, CY['l1'], -W['l1'] / 2, CY['l1'])

    fig.set_size_inches(xmax - xmin, ymax - ymin)
    fig.savefig('../nnh_tw_main.png', dpi=200, facecolor='white')
    print('nnh_tw_main.png saved')


# ----------------------------------------------------------------------------
#  Рисунок 2 — подпрограмма «Выбрать следующую задачу»
# ----------------------------------------------------------------------------
def build_select():
    GS = 0.46  # более плотный шаг для компактной подпрограммы
    D = {
        'start':  ('terminal',  'Выбрать следующую задачу', 9),
        'reset':  ('process',   'Сбросить выбор\nи лучшую оценку', FS),
        'p1':     ('decision',  'Проход 1: остался\nнепросмотренный кандидат?', FS),
        'prec1':  ('decision',  'Обязательные\nпредшественники\nзадачи выполнены?', FS),
        'feas':   ('decision',  'Успеваем\nв ее временное окно?', FS),
        'eval':   ('process',   'Оценить: когда освободимся\nи не сорвем ли окна других\nзадач; если лучше — запомнить', FS),
        'found':  ('decision',  'Допустимая\nзадача найдена?', FS),
        'p2':     ('decision',  'Проход 2: остался\nнепросмотренный кандидат?', FS),
        'prec2':  ('decision',  'Обязательные\nпредшественники\nзадачи выполнены?', FS),
        'cmp':    ('process',   'Если время завершения\nменьше лучшего —\nзапомнить кандидата', FS),
        'ret':    ('terminal',  'Возврат: выбранная задача', 9),
    }
    W, H, KIND = measure(D)
    spine = ['start', 'reset', 'p1', 'prec1', 'feas', 'eval', 'found',
             'p2', 'prec2', 'cmp', 'ret']
    CY, X = {}, {k: 0.0 for k in D}
    y = 0.0
    for k in spine:
        CY[k] = y - H[k] / 2
        y -= H[k] + GS

    def top(k):
        return CY[k] + H[k] / 2

    def bot(k):
        return CY[k] - H[k] / 2

    spine_hw = max(W[k] for k in spine) / 2
    r1 = spine_hw + 1.2          # возврат тела прохода 1 -> p1
    r2 = spine_hw + 1.2          # возврат тела прохода 2 -> p2
    lL = -(spine_hw + 1.2)       # «нет» p1/p2 -> обход тела
    rF = spine_hw + 2.2          # found «да» -> ret

    xmin = lL - 0.5
    xmax = rF + 0.5
    ymax = 0.4
    ymin = bot('ret') - 0.5

    fig = plt.figure(dpi=DPI)
    ax = fig.add_axes([0, 0, 1, 1])
    ax.set_aspect('equal')
    ax.axis('off')
    ax.set_xlim(xmin, xmax)
    ax.set_ylim(ymin, ymax)

    for k in D:
        DRAW[KIND[k]](ax, X[k], CY[k], W[k], H[k], D[k][1], fs=D[k][2])

    # спайн прохода 1: p1 -> prec1 -> feas -> eval
    arrow(ax, 0, bot('start'), 0, top('reset'))
    arrow(ax, 0, bot('reset'), 0, top('p1'))
    arrow(ax, 0, bot('p1'), 0, top('prec1'))
    tag(ax, 0.12, bot('p1') - 0.18, 'да')
    arrow(ax, 0, bot('prec1'), 0, top('feas'))
    tag(ax, 0.12, bot('prec1') - 0.18, 'да')
    arrow(ax, 0, bot('feas'), 0, top('eval'))
    tag(ax, 0.12, bot('feas') - 0.18, 'да')

    # обратные дуги прохода 1 (prec1 «нет», feas «нет», конец eval) -> r1 -> p1
    line(ax, [(0, bot('eval')), (0, bot('eval') - 0.32), (r1, bot('eval') - 0.32)])
    line(ax, [(W['prec1'] / 2, CY['prec1']), (r1, CY['prec1'])])
    tag(ax, W['prec1'] / 2 + 0.1, CY['prec1'] + 0.16, 'нет')
    line(ax, [(W['feas'] / 2, CY['feas']), (r1, CY['feas'])])
    tag(ax, W['feas'] / 2 + 0.1, CY['feas'] + 0.16, 'нет')
    line(ax, [(r1, bot('eval') - 0.32), (r1, CY['p1'])])
    arrow(ax, r1, CY['p1'], W['p1'] / 2, CY['p1'])

    # p1 «нет» -> левый рельс -> found
    line(ax, [(-W['p1'] / 2, CY['p1']), (lL, CY['p1']), (lL, CY['found'])])
    arrow(ax, lL, CY['found'], -W['found'] / 2, CY['found'])
    tag(ax, -W['p1'] / 2 - 0.12, CY['p1'] + 0.18, 'нет', ha='right')

    # found «нет» -> проход 2 (p2 -> prec2 -> cmp)
    arrow(ax, 0, bot('found'), 0, top('p2'))
    tag(ax, 0.12, bot('found') - 0.18, 'нет')
    arrow(ax, 0, bot('p2'), 0, top('prec2'))
    tag(ax, 0.12, bot('p2') - 0.18, 'да')
    arrow(ax, 0, bot('prec2'), 0, top('cmp'))
    tag(ax, 0.12, bot('prec2') - 0.18, 'да')

    # обратные дуги прохода 2 (prec2 «нет», конец cmp) -> r2 -> p2
    line(ax, [(0, bot('cmp')), (0, bot('cmp') - 0.32), (r2, bot('cmp') - 0.32)])
    line(ax, [(W['prec2'] / 2, CY['prec2']), (r2, CY['prec2'])])
    tag(ax, W['prec2'] / 2 + 0.1, CY['prec2'] + 0.16, 'нет')
    line(ax, [(r2, bot('cmp') - 0.32), (r2, CY['p2'])])
    arrow(ax, r2, CY['p2'], W['p2'] / 2, CY['p2'])

    # found «да» -> дальний рельс rF -> вниз -> ret
    line(ax, [(W['found'] / 2, CY['found']), (rF, CY['found']), (rF, CY['ret'])])
    arrow(ax, rF, CY['ret'], W['ret'] / 2, CY['ret'])
    tag(ax, W['found'] / 2 + 0.12, CY['found'] + 0.18, 'да')

    # p2 «нет» -> левый рельс -> ret
    line(ax, [(-W['p2'] / 2, CY['p2']), (lL, CY['p2']), (lL, CY['ret'])])
    arrow(ax, lL, CY['ret'], -W['ret'] / 2, CY['ret'])
    tag(ax, -W['p2'] / 2 - 0.12, CY['p2'] + 0.18, 'нет', ha='right')

    fig.set_size_inches(xmax - xmin, ymax - ymin)
    fig.savefig('../nnh_tw_select.png', dpi=200, facecolor='white')
    print('nnh_tw_select.png saved')


build_main()
build_select()
