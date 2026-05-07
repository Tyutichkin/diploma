import { Outlet, Link, useLocation } from 'react-router';
import { Button } from './ui/button';
import { LogOut, BookOpen, LayoutDashboard } from 'lucide-react';
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
        <div className="container mx-auto px-3 sm:px-4 py-2 sm:py-3 max-w-full">
          <div className="flex items-center justify-between gap-2 sm:gap-3">
            <div className="min-w-0 flex-1">
              <h1 className="text-base sm:text-xl lg:text-2xl font-bold text-gray-900 leading-tight truncate">
                Планировщик задач
              </h1>
              <p className="hidden md:block text-xs lg:text-sm text-gray-600 leading-tight truncate">
                Оптимизация маршрутов с учётом временных окон
              </p>
            </div>
            <div className="flex items-center gap-2 sm:gap-3 lg:gap-4 flex-shrink-0 min-w-0">
              <span
                className="hidden md:inline text-xs lg:text-sm text-gray-600 truncate max-w-[140px] lg:max-w-[220px]"
                title={session?.email}
              >
                {session?.email}
              </span>
              <Button variant="outline" size="sm" onClick={() => void logout()} className="flex-shrink-0">
                <LogOut className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">Выйти</span>
                <span className="sr-only sm:hidden">Выйти</span>
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white border-b">
        <div className="container mx-auto px-3 sm:px-4 max-w-full">
          <div className="flex gap-1">
            <Link to="/">
              <Button
                variant="ghost"
                size="sm"
                className={`rounded-none border-b-2 h-9 ${
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
                size="sm"
                className={`rounded-none border-b-2 h-9 ${
                  isActive('/instructions')
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                <BookOpen className="mr-2 h-4 w-4" />
                Инструкция
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
