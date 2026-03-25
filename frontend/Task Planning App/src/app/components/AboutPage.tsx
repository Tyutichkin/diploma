import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { 
  User, 
  Mail, 
  Github, 
  Linkedin,
  Globe,
  Code,
  Sparkles,
  Heart
} from 'lucide-react';

export function AboutPage() {
  return (
    <div className="container mx-auto px-4 py-8 max-w-4xl">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2 flex items-center gap-3">
          <User className="h-8 w-8 text-blue-600" />
          Об авторе
        </h1>
        <p className="text-gray-600">
          Информация о разработчике приложения
        </p>
      </div>

      <div className="space-y-6">
        {/* Основная информация */}
        <Card className="bg-gradient-to-br from-blue-50 to-indigo-50 border-blue-200">
          <CardContent className="pt-6">
            <div className="flex flex-col md:flex-row gap-6 items-start">
              <div className="flex-shrink-0">
                <div className="w-32 h-32 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-full flex items-center justify-center">
                  <User className="h-16 w-16 text-white" />
                </div>
              </div>
              <div className="flex-1">
                <h2 className="text-2xl font-bold text-gray-900 mb-2">Разработчик приложения</h2>
                <p className="text-gray-700 mb-4">
                  Веб-разработчик, специализирующийся на создании современных 
                  веб-приложений с использованием React, TypeScript и современных технологий.
                </p>
                <div className="flex flex-wrap gap-2">
                  <span className="px-3 py-1 bg-blue-100 text-blue-700 rounded-full text-sm font-medium">
                    React
                  </span>
                  <span className="px-3 py-1 bg-indigo-100 text-indigo-700 rounded-full text-sm font-medium">
                    TypeScript
                  </span>
                  <span className="px-3 py-1 bg-purple-100 text-purple-700 rounded-full text-sm font-medium">
                    Tailwind CSS
                  </span>
                  <span className="px-3 py-1 bg-green-100 text-green-700 rounded-full text-sm font-medium">
                    Leaflet Maps
                  </span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Технологии проекта */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Code className="h-5 w-5 text-purple-600" />
              Технологический стек
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-3">
                <div>
                  <h3 className="font-semibold text-gray-900 mb-1">Frontend</h3>
                  <ul className="list-disc list-inside text-gray-700 space-y-1 ml-2">
                    <li>React 18</li>
                    <li>TypeScript</li>
                    <li>React Router</li>
                    <li>React DnD (Drag & Drop)</li>
                  </ul>
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900 mb-1">Стилизация</h3>
                  <ul className="list-disc list-inside text-gray-700 space-y-1 ml-2">
                    <li>Tailwind CSS v4</li>
                    <li>Radix UI компоненты</li>
                    <li>Lucide иконки</li>
                  </ul>
                </div>
              </div>
              <div className="space-y-3">
                <div>
                  <h3 className="font-semibold text-gray-900 mb-1">Карты и геокодинг</h3>
                  <ul className="list-disc list-inside text-gray-700 space-y-1 ml-2">
                    <li>Leaflet.js</li>
                    <li>OpenStreetMap</li>
                    <li>Nominatim API</li>
                  </ul>
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900 mb-1">Инструменты</h3>
                  <ul className="list-disc list-inside text-gray-700 space-y-1 ml-2">
                    <li>Vite</li>
                    <li>Sonner (уведомления)</li>
                  </ul>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Возможности приложения */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-yellow-600" />
              Ключевые возможности
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-3 text-gray-700">
              <li className="flex items-start gap-3">
                <div className="flex-shrink-0 w-6 h-6 bg-green-100 rounded-full flex items-center justify-center mt-0.5">
                  <span className="text-green-600 text-sm">✓</span>
                </div>
                <div>
                  <strong>Геокодирование адресов</strong> — автоматическое определение координат 
                  по адресу с помощью Nominatim API
                </div>
              </li>
              <li className="flex items-start gap-3">
                <div className="flex-shrink-0 w-6 h-6 bg-green-100 rounded-full flex items-center justify-center mt-0.5">
                  <span className="text-green-600 text-sm">✓</span>
                </div>
                <div>
                  <strong>Оптимизация маршрута</strong> — алгоритм построения оптимального 
                  пути с учётом временных окон
                </div>
              </li>
              <li className="flex items-start gap-3">
                <div className="flex-shrink-0 w-6 h-6 bg-green-100 rounded-full flex items-center justify-center mt-0.5">
                  <span className="text-green-600 text-sm">✓</span>
                </div>
                <div>
                  <strong>Интерактивная карта</strong> — визуализация маршрута на карте 
                  OpenStreetMap с маркерами задач
                </div>
              </li>
              <li className="flex items-start gap-3">
                <div className="flex-shrink-0 w-6 h-6 bg-green-100 rounded-full flex items-center justify-center mt-0.5">
                  <span className="text-green-600 text-sm">✓</span>
                </div>
                <div>
                  <strong>Drag & Drop интерфейс</strong> — удобное управление порядком 
                  задач перетаскиванием
                </div>
              </li>
              <li className="flex items-start gap-3">
                <div className="flex-shrink-0 w-6 h-6 bg-green-100 rounded-full flex items-center justify-center mt-0.5">
                  <span className="text-green-600 text-sm">✓</span>
                </div>
                <div>
                  <strong>Экспорт данных</strong> — возможность экспорта маршрута в формате JSON
                </div>
              </li>
              <li className="flex items-start gap-3">
                <div className="flex-shrink-0 w-6 h-6 bg-green-100 rounded-full flex items-center justify-center mt-0.5">
                  <span className="text-green-600 text-sm">✓</span>
                </div>
                <div>
                  <strong>Responsive дизайн</strong> — адаптивный интерфейс для всех устройств
                </div>
              </li>
            </ul>
          </CardContent>
        </Card>

        {/* Контактная информация */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Mail className="h-5 w-5 text-blue-600" />
              Контакты
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <p className="text-gray-700">
                Если у вас есть вопросы, предложения или вы хотите обсудить проект, 
                свяжитесь со мной любым удобным способом:
              </p>
              <div className="flex flex-wrap gap-4">
                <a 
                  href="mailto:developer@example.com" 
                  className="flex items-center gap-2 px-4 py-2 bg-blue-50 hover:bg-blue-100 text-blue-700 rounded-lg transition-colors"
                >
                  <Mail className="h-4 w-4" />
                  <span>developer@example.com</span>
                </a>
                <a 
                  href="https://github.com" 
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 px-4 py-2 bg-gray-50 hover:bg-gray-100 text-gray-700 rounded-lg transition-colors"
                >
                  <Github className="h-4 w-4" />
                  <span>GitHub</span>
                </a>
                <a 
                  href="https://linkedin.com" 
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 px-4 py-2 bg-blue-50 hover:bg-blue-100 text-blue-700 rounded-lg transition-colors"
                >
                  <Linkedin className="h-4 w-4" />
                  <span>LinkedIn</span>
                </a>
                <a 
                  href="https://example.com" 
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 px-4 py-2 bg-purple-50 hover:bg-purple-100 text-purple-700 rounded-lg transition-colors"
                >
                  <Globe className="h-4 w-4" />
                  <span>Портфолио</span>
                </a>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Footer */}
        <Card className="bg-gradient-to-br from-pink-50 to-rose-50 border-pink-200">
          <CardContent className="pt-6 text-center">
            <div className="flex items-center justify-center gap-2 text-gray-700">
              <Heart className="h-5 w-5 text-red-500 fill-red-500" />
              <p>Сделано с любовью к веб-разработке</p>
            </div>
            <p className="text-sm text-gray-600 mt-2">
              © 2026 Планировщик задач. Все права защищены.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
