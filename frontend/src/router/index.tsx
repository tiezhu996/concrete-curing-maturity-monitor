import { Navigate, createBrowserRouter, useLocation } from 'react-router-dom'
import { AppShell } from '../components/common/AppShell'
import { useAuth } from '../hooks/useAuth'
import { LoginPage } from '../pages/LoginPage'
import { SectionsPage } from '../pages/SectionsPage'
import { MixDesignsPage } from '../pages/MixDesignsPage'
import { TemperaturesPage } from '../pages/TemperaturesPage'
import { ForecastsPage } from '../pages/ForecastsPage'
import { AuditPage } from '../pages/AuditPage'

function ProtectedShell() {
  const { authenticated } = useAuth()
  const location = useLocation()
  return authenticated ? <AppShell /> : <Navigate to="/login" replace state={{ from: location.pathname }} />
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/', element: <ProtectedShell />, children: [
      { index: true, element: <Navigate to="/sections" replace /> },
      { path: 'sections', element: <SectionsPage /> },
      { path: 'mix-designs', element: <MixDesignsPage /> },
      { path: 'temperatures', element: <TemperaturesPage /> },
      { path: 'forecasts', element: <ForecastsPage /> },
      { path: 'audit', element: <AuditPage /> },
    ],
  },
  { path: '*', element: <Navigate to="/sections" replace /> },
])
