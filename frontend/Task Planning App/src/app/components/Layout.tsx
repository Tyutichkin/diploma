import { Outlet, Link, useLocation } from 'react-router';
import { Button } from './ui/button';
import { LogOut, BookOpen, User, LayoutDashboard } from 'lucide-react';
import { useAuth } from '../context/auth-context';

export function Layout() {
  const location = useLocation();
  const { session, logout } = useAuth();

  const isActive = (path: string) => {
    return location.pathname === path;
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b shadow-sm sticky top-0 z-40">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Планировщик задач</h1>
              <p className="text-sm text-gray-600">Оптимизация маршрутов с учётом временных окон</p>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-sm text-gray-600 hidden sm:inline">{session?.email}</span>
              <Button variant="outline" size="sm" onClick={() => void logout()}>
                <LogOut className="mr-2 h-4 w-4" />
                Выйти
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white border-b">
        <div className="container mx-auto px-4">
          <div className="flex gap-1">
            <Link to="/">
              <Button 
                variant="ghost" 
                className={`rounded-none border-b-2 ${
                  isActive('/') 
                    ? 'border-blue-600 text-blue-600' 
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                <LayoutDashboard className="mr-2 h-4 w-4" />
                Главная
              </Button>
            </Link>
            <Link to="/instructions">
              <Button 
                variant="ghost"
                className={`rounded-none border-b-2 ${
                  isActive('/instructions') 
                    ? 'border-blue-600 text-blue-600' 
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                <BookOpen className="mr-2 h-4 w-4" />
                Инструкция
              </Button>
            </Link>
            <Link to="/about">
              <Button 
                variant="ghost"
                className={`rounded-none border-b-2 ${
                  isActive('/about') 
                    ? 'border-blue-600 text-blue-600' 
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                <User className="mr-2 h-4 w-4" />
                Об авторе
              </Button>
            </Link>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main>
        <Outlet />
      </main>
    </div>
  );
}
