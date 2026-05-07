import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import ToastContainer from './components/Toast'
import Layout from './components/Layout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import LogViewer from './pages/LogViewer'
import Settings from './pages/Settings'

function ProtectedRoute({ children }) {
  const token = localStorage.getItem('loghub_token')
  if (!token) {
    return <Navigate to="/login" replace />
  }
  return children
}

function PublicRoute({ children }) {
  const token = localStorage.getItem('loghub_token')
  if (token) {
    return <Navigate to="/" replace />
  }
  return children
}

export default function App() {
  return (
    <BrowserRouter>
      <ToastContainer />
      <Routes>
        <Route
          path="/login"
          element={
            <PublicRoute>
              <Login />
            </PublicRoute>
          }
        />
        <Route
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route path="/" element={<Dashboard />} />
          <Route path="/logs/:appId" element={<LogViewer />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
