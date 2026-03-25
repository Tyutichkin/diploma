import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { 
  BookOpen, 
  MapPin, 
  Route, 
  Download, 
  Clock,
  Info,
  CheckCircle2,
  Plus,
  GripVertical
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
        {/* Введение */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Info className="h-5 w-5 text-blue-600" />
              О приложении
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>
              Планировщик задач — это приложение для автоматизированного планирования 
              маршрутов с учётом географического расположения задач и временных ограничений.
            </p>
            <p>
              Приложение помогает оптимизировать маршрут выполнения задач, минимизируя 
              время в пути и учитывая временные окна для каждой задачи.
            </p>
          </CardContent>
        </Card>

        {/* Добавление задач */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Plus className="h-5 w-5 text-green-600" />
              Шаг 1: Добавление задач
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <ol className="list-decimal list-inside space-y-2 ml-2">
              <li>Нажмите кнопку <strong>"Добавить задачу"</strong> в верхней части страницы</li>
              <li>Заполните обязательные поля:
                <ul className="list-disc list-inside ml-6 mt-1 space-y-1">
                  <li><strong>Название задачи</strong> — краткое описание</li>
                  <li><strong>Адрес</strong> — место выполнения задачи</li>
                  <li><strong>Длительность</strong> — время выполнения в минутах</li>
                </ul>
              </li>
              <li>Опционально укажите временное окно:
                <ul className="list-disc list-inside ml-6 mt-1 space-y-1">
                  <li><strong>Начало окна</strong> — когда можно начать задачу</li>
                  <li><strong>Конец окна</strong> — когда задача должна быть завершена</li>
                </ul>
              </li>
              <li>Нажмите <strong>"Добавить"</strong> для сохранения задачи</li>
            </ol>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 mt-4">
              <p className="text-sm text-blue-900">
                <strong>Подсказка:</strong> Адреса автоматически геокодируются для точного 
                отображения на карте. Старайтесь указывать полные адреса.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Управление списком */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GripVertical className="h-5 w-5 text-purple-600" />
              Шаг 2: Управление списком задач
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>В списке задач вы можете:</p>
            <ul className="list-disc list-inside space-y-2 ml-2">
              <li><strong>Изменить порядок</strong> — перетащите задачу, удерживая её</li>
              <li><strong>Редактировать</strong> — нажмите на задачу для редактирования</li>
              <li><strong>Удалить</strong> — используйте иконку корзины</li>
              <li><strong>Отметить как выполненную</strong> — поставьте галочку</li>
            </ul>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 mt-4">
              <p className="text-sm text-blue-900">
                <strong>Подсказка:</strong> Используйте drag-and-drop для ручной сортировки 
                задач перед оптимизацией.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Оптимизация маршрута */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Route className="h-5 w-5 text-blue-600" />
              Шаг 3: Оптимизация маршрута
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <ol className="list-decimal list-inside space-y-2 ml-2">
              <li>Убедитесь, что добавлено минимум 2 задачи</li>
              <li>Нажмите кнопку <strong>"Оптимизировать маршрут"</strong></li>
              <li>Алгоритм автоматически:
                <ul className="list-disc list-inside ml-6 mt-1 space-y-1">
                  <li>Рассчитает расстояния между точками</li>
                  <li>Учтёт временные окна задач</li>
                  <li>Минимизирует общее время в пути</li>
                  <li>Построит оптимальный маршрут</li>
                </ul>
              </li>
              <li>После оптимизации задачи будут переупорядочены автоматически</li>
            </ol>
            <div className="bg-green-50 border border-green-200 rounded-lg p-3 mt-4">
              <p className="text-sm text-green-900">
                <strong>Результат:</strong> Вы увидите информационную панель с данными о 
                маршруте (расстояние, время в пути, общее время).
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Работа с картой */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MapPin className="h-5 w-5 text-red-600" />
              Шаг 4: Работа с картой
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <p>Интерактивная карта позволяет:</p>
            <ul className="list-disc list-inside space-y-2 ml-2">
              <li><strong>Просматривать маршрут</strong> — визуализация всех задач на карте</li>
              <li><strong>Нажимать на маркеры</strong> — просмотр детальной информации о задаче</li>
              <li><strong>Масштабировать</strong> — увеличение/уменьшение карты</li>
              <li><strong>Перемещать</strong> — навигация по карте</li>
            </ul>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 mt-4">
              <p className="text-sm text-blue-900">
                <strong>Обратите внимание:</strong> После оптимизации маршрут отображается 
                синей линией, соединяющей задачи в оптимальном порядке.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Экспорт данных */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Download className="h-5 w-5 text-indigo-600" />
              Шаг 5: Экспорт маршрута
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-gray-700">
            <ol className="list-decimal list-inside space-y-2 ml-2">
              <li>Нажмите кнопку <strong>"Экспорт в JSON"</strong></li>
              <li>Файл будет автоматически загружен на ваш компьютер</li>
              <li>В файле содержится:
                <ul className="list-disc list-inside ml-6 mt-1 space-y-1">
                  <li>Список всех задач с координатами</li>
                  <li>Порядок выполнения</li>
                  <li>Временные окна</li>
                  <li>Статистика маршрута (если оптимизирован)</li>
                </ul>
              </li>
            </ol>
          </CardContent>
        </Card>

        {/* Советы */}
        <Card className="bg-gradient-to-br from-blue-50 to-indigo-50 border-blue-200">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-blue-600" />
              Полезные советы
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-gray-700">
            <ul className="list-disc list-inside space-y-2 ml-2">
              <li>Указывайте реалистичные временные окна для лучшей оптимизации</li>
              <li>Проверяйте координаты задач на карте после добавления</li>
              <li>Используйте drag-and-drop для быстрой корректировки порядка</li>
              <li>Экспортируйте маршрут для использования в других приложениях</li>
              <li>Для точности геокодирования указывайте полные адреса с городом</li>
            </ul>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
