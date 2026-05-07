import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import {
  AlertTriangle,
  ArrowDownUp,
  BookOpen,
  CheckCircle2,
  Download,
  Flag,
  GripVertical,
  Info,
  ListOrdered,
  MapPin,
  Plus,
  Route,
  Upload,
} from 'lucide-react';

export function InstructionsPage() {
  return (
    <div className="container mx-auto px-4 py-8 max-w-4xl">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2 flex items-center gap-3">
          <BookOpen className="h-8 w-8 text-blue-600" />
          Инструкция по использованию
        </h1>
        <p className="text-gray-600">
          Руководство по работе с планировщиком задач и оптимизатором маршрутов
        </p>
      </div>

      <div className="space-y-6">
        {/* О приложении */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Info className="h-5 w-5 text-blue-600" />
              О приложении
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>
              Планировщик задач помогает спланировать день по списку задач,
              привязанных к адресам: приложение само строит порядок их объезда
              с учётом времени в пути, длительности выполнения и временных окон.
            </p>
            <p>
              Маршрут строится на стороне сервера (алгоритм NNH-TW) по матрице
              расстояний, полученной через Яндекс&nbsp;Карты, и сохраняется
              в учётной записи пользователя — на каждого пользователя хранится
              один последний построенный маршрут.
            </p>
          </CardContent>
        </Card>

        {/* Шаг 1 — Добавление задач */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Plus className="h-5 w-5 text-green-600" />
              Шаг 1: Добавление задач
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <ol className="list-decimal list-inside space-y-2 ml-2">
              <li>
                Нажмите кнопку <strong>«Добавить задачу»</strong> в верхней
                панели инструментов.
              </li>
              <li>
                Заполните поля формы:
                <ul className="list-disc list-inside ml-6 mt-1 space-y-1">
                  <li>
                    <strong>Название</strong> — обязательное поле.
                  </li>
                  <li>
                    <strong>Адрес</strong> — необязательный. По мере набора
                    появляются подсказки от Яндекс&nbsp;Геокодера; выбор
                    подсказки фиксирует координаты автоматически.
                  </li>
                  <li>
                    <strong>Длительность (мин)</strong> — необязательное поле,
                    время выполнения задачи в минутах.
                  </li>
                  <li>
                    <strong>Временное окно</strong> — даты и время начала
                    и окончания. Каждое поле можно очистить отдельно;
                    допустимо указать только время или только дату.
                  </li>
                </ul>
              </li>
              <li>
                При желании сразу назначьте задачу начальной или конечной
                точкой маршрута переключателем в форме.
              </li>
              <li>
                Нажмите <strong>«Сохранить»</strong> — задача появится в списке.
              </li>
            </ol>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 mt-4">
              <p className="text-sm text-blue-900">
                <strong>Подсказка:</strong> задачи без адреса тоже сохраняются —
                они попадают в отдельную группу под основным списком и не
                участвуют в оптимизации маршрута, но позволяют вести общий
                перечень дел.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Шаг 2 — Импорт из файла */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Upload className="h-5 w-5 text-emerald-600" />
              Шаг 2: Импорт задач из файла
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>
              Кнопка <strong>«Импорт из файла»</strong> открывает диалог
              загрузки списка задач из CSV или Excel-файла (.xlsx).
            </p>
            <p>Ожидаемые колонки заголовка:</p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <code className="px-1 rounded bg-gray-100">title</code> —
                название (обязательно).
              </li>
              <li>
                <code className="px-1 rounded bg-gray-100">address</code> —
                адрес для геокодирования.
              </li>
              <li>
                <code className="px-1 rounded bg-gray-100">duration</code> —
                длительность в минутах.
              </li>
              <li>
                <code className="px-1 rounded bg-gray-100">date</code> — дата
                в формате <code>ДД.ММ.ГГГГ</code> или <code>ГГГГ-ММ-ДД</code>.
                Если пусто — задача попадает на сегодня.
              </li>
              <li>
                <code className="px-1 rounded bg-gray-100">window_start</code>,{' '}
                <code className="px-1 rounded bg-gray-100">window_end</code> —
                время начала и окончания окна в формате <code>ЧЧ:ММ</code>.
              </li>
            </ul>
            <p>
              После загрузки приложение автоматически геокодирует адреса.
              Строки с ошибочными значениями будут показаны отдельным списком,
              корректные задачи добавляются в конец основного списка.
            </p>
          </CardContent>
        </Card>

        {/* Шаг 3 — Управление списком задач */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GripVertical className="h-5 w-5 text-purple-600" />
              Шаг 3: Управление списком задач
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>На карточке задачи и в её контекстном меню доступны действия:</p>
            <ul className="list-disc list-inside space-y-2 ml-2">
              <li>
                <strong>Изменить порядок</strong> — перетащите карточку,
                удерживая её за иконку слева. Адресные и безадресные задачи
                перетаскиваются в пределах своих групп.
              </li>
              <li>
                <strong>Редактировать</strong> — кнопка с карандашом на карточке.
              </li>
              <li>
                <strong>Отметить выполненной</strong> — флажок слева; статус
                сохраняется в учётной записи и переживает перезаход в систему.
              </li>
              <li>
                <strong>Клонировать</strong> — пункт «Клонировать» в выпадающем
                меню; копия добавляется в конец списка.
              </li>
              <li>
                <strong>Удалить одну задачу</strong> — кнопка с корзиной;
                требуется подтверждение.
              </li>
              <li>
                <strong>Очистить всё</strong> — кнопка над списком удаляет все
                задачи пользователя; ранее построенный маршрут также сбрасывается.
              </li>
            </ul>
            <div className="bg-amber-50 border border-amber-200 rounded-lg p-3 mt-4">
              <p className="text-sm text-amber-900 flex gap-2 items-start">
                <AlertTriangle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                <span>
                  Если временные окна задач пересекаются или маршрут не
                  укладывается в окно, соответствующие карточки подсвечиваются.
                  Конфликты нужно устранить до запуска оптимизации.
                </span>
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Шаг 4 — Начальная и конечная точки */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Flag className="h-5 w-5 text-rose-600" />
              Шаг 4: Начальная и конечная точки маршрута
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>
              По умолчанию алгоритм сам выбирает первую и последнюю точку.
              Если требуется зафиксировать стартовую или финишную задачу
              (например, начать день из дома, а закончить в офисе):
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                Откройте контекстное меню задачи (иконка с тремя точками или
                правый клик).
              </li>
              <li>
                Выберите <strong>«Назначить начальной точкой»</strong> или{' '}
                <strong>«Назначить конечной точкой»</strong>. Та же опция
                доступна прямо в форме редактирования задачи.
              </li>
              <li>
                Чтобы снять закрепление — выберите ту же команду повторно.
              </li>
            </ul>
            <p>
              Закреплённые точки помечаются флажком на карточке и учитываются
              при следующем запуске оптимизации.
            </p>
          </CardContent>
        </Card>

        {/* Шаг 5 — Ограничения порядка */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ArrowDownUp className="h-5 w-5 text-indigo-600" />
              Шаг 5: Ограничения порядка выполнения
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>
              Если в списке две и более задач, над основной областью появляется
              панель <strong>«Ограничения порядка выполнения задач»</strong>.
              Она позволяет задавать парные правила вида «задача&nbsp;A
              выполняется раньше задачи&nbsp;B».
            </p>
            <ol className="list-decimal list-inside space-y-1 ml-2">
              <li>
                Нажмите <strong>«Добавить»</strong> — появится новая пара
                с двумя выпадающими списками.
              </li>
              <li>Выберите задачу&nbsp;A (раньше) и задачу&nbsp;B (позже).</li>
              <li>
                Повторите для нужного количества правил; неполные или
                совпадающие пары игнорируются.
              </li>
              <li>
                Удалить правило можно крестиком справа от строки.
              </li>
            </ol>
            <p>
              Указанные ограничения учитываются алгоритмом наряду с временными
              окнами.
            </p>
          </CardContent>
        </Card>

        {/* Шаг 6 — Оптимизация */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Route className="h-5 w-5 text-blue-600" />
              Шаг 6: Оптимизация маршрута
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <ol className="list-decimal list-inside space-y-2 ml-2">
              <li>
                Убедитесь, что в списке не менее двух задач с заполненным
                адресом и нет конфликтов временных окон.
              </li>
              <li>
                Нажмите кнопку <strong>«Оптимизировать маршрут»</strong>.
              </li>
              <li>
                Приложение последовательно:
                <ul className="list-disc list-inside ml-6 mt-1 space-y-1">
                  <li>
                    запросит у Яндекс&nbsp;Карт матрицу расстояний и времени
                    в пути для выбранного режима передвижения;
                  </li>
                  <li>
                    отправит её на сервер, который вычислит оптимальный порядок
                    с учётом временных окон, начальной и конечной точки,
                    ограничений порядка;
                  </li>
                  <li>
                    переупорядочит карточки и сохранит результат в учётной
                    записи.
                  </li>
                </ul>
              </li>
              <li>
                Над списком появится сводная панель: суммарное расстояние,
                время в пути и общее время с учётом выполнения задач.
              </li>
            </ol>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 mt-4">
              <p className="text-sm text-blue-900">
                <strong>Важно:</strong> на каждого пользователя хранится только
                один маршрут. Каждый новый запуск оптимизации перезаписывает
                предыдущий результат.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Шаг 7 — Карта */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MapPin className="h-5 w-5 text-red-600" />
              Шаг 7: Работа с картой
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>На карте отображаются все задачи с координатами:</p>
            <ul className="list-disc list-inside space-y-2 ml-2">
              <li>
                <strong>Пронумерованные метки</strong> соответствуют порядку
                выполнения. Совпадающие координаты группируются в стопку, чтобы
                метки не перекрывались.
              </li>
              <li>
                <strong>Переключатель режима передвижения</strong> в верхней
                части карты — «Пешком», «Транспорт», «Авто». Смена режима
                автоматически перестраивает уже построенный маршрут с учётом
                новой матрицы расстояний.
              </li>
              <li>
                <strong>Маршрут после оптимизации</strong> отрисовывается
                сегментами с разной стилистикой линии в зависимости от способа
                передвижения; в режиме «Транспорт» рядом с сегментами выводятся
                плашки с номерами автобусов, маршрутов метро и т. п.
              </li>
              <li>
                Клик по метке показывает название и адрес задачи; колесом мыши
                и жестами доступны привычные масштабирование и панорамирование.
              </li>
            </ul>
          </CardContent>
        </Card>

        {/* Шаг 8 — Список шагов маршрута */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ListOrdered className="h-5 w-5 text-blue-600" />
              Шаг 8: Список шагов маршрута
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>
              Под картой и списком задач после оптимизации появляется
              детализированный список шагов. Для каждой остановки выводятся:
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>время прибытия и завершения задачи;</li>
              <li>длительность переезда от предыдущей точки;</li>
              <li>
                ожидание, если приехали раньше открытия временного окна;
              </li>
              <li>
                для режима «Транспорт» — конкретные маршруты, остановки посадки
                и высадки.
              </li>
            </ul>
            <p>
              Если у задачи нарушено временное окно (например, выполнение
              заканчивается позже допустимого), карточка шага помечается значком{' '}
              <AlertTriangle className="inline h-4 w-4 text-amber-600 align-text-bottom" />{' '}
              с пояснением. Из списка шагов можно сразу отредактировать или
              удалить задачу.
            </p>
          </CardContent>
        </Card>

        {/* Шаг 9 — Экспорт */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Download className="h-5 w-5 text-indigo-600" />
              Шаг 9: Экспорт задач и маршрута
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>
              Кнопка <strong>«Экспорт»</strong> открывает диалог выбора формата:
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <strong>CSV</strong> — текстовый файл со столбцами в том же
                формате, что и для импорта. Подходит для последующей загрузки
                обратно в приложение.
              </li>
              <li>
                <strong>Excel (.xlsx)</strong> — то же содержимое в виде
                табличной книги Excel.
              </li>
              <li>
                <strong>JSON</strong> — полная выгрузка задач и параметров
                построенного маршрута, включая порядок остановок, времена
                прибытия и общую статистику.
              </li>
            </ul>
            <p>
              Файл сохраняется штатным механизмом загрузки браузера.
            </p>
          </CardContent>
        </Card>

        {/* Полезные советы */}
        <Card className="bg-gradient-to-br from-blue-50 to-indigo-50 border-blue-200">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-blue-600" />
              Полезные советы
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-gray-700">
            <ul className="list-disc list-inside space-y-2 ml-2">
              <li>
                Указывайте полный адрес с городом и улицей — это повышает
                точность геокодирования и снижает количество подсказок.
              </li>
              <li>
                Реалистичные временные окна и длительности позволяют алгоритму
                строить более устойчивые маршруты и заранее обнаруживать
                конфликты.
              </li>
              <li>
                Сначала зафиксируйте начальную и конечную точки и ограничения
                порядка, и только потом запускайте оптимизацию.
              </li>
              <li>
                Если планируется ежедневная перезагрузка списка — храните задачи
                в CSV/Excel и импортируйте их одной кнопкой.
              </li>
              <li>
                Для смены режима передвижения достаточно переключателя на карте —
                полностью пересчитывать маршрут вручную не нужно.
              </li>
            </ul>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
