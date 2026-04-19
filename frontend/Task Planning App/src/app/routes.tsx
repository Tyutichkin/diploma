import { createBrowserRouter } from 'react-router';
import { Layout } from './components/Layout';
import { MainPage } from './components/MainPage';
import { InstructionsPage } from './components/InstructionsPage';

export function createRouter() {
  return createBrowserRouter([
    {
      path: '/',
      element: <Layout />,
      children: [
        {
          index: true,
          element: <MainPage />,
        },
        {
          path: 'instructions',
          element: <InstructionsPage />,
        },
      ],
    },
  ]);
}
