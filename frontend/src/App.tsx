import { HashRouter, Routes, Route, useLocation, Navigate } from 'react-router-dom'
import { AnimatePresence, motion } from 'framer-motion'
import type { ReactNode } from 'react'
import LoginScreen from './pages/LoginScreen'
import ConnectScreen from './pages/ConnectScreen'
import BroadcastScreen from './pages/BroadcastScreen'
import { ToastProvider } from './components/ToastProvider'
import { getToken, isAdmin } from './lib/api'
import './App.css'

function RequireAuth({ children }: { children: ReactNode }) {
  if (!getToken()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function RequireAdmin({ children }: { children: ReactNode }) {
  if (!getToken()) {
    return <Navigate to="/login" replace />
  }
  if (!isAdmin()) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

function Page({ children }: { children: ReactNode }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -12 }}
      transition={{ duration: 0.25, ease: 'easeOut' }}
    >
      {children}
    </motion.div>
  )
}

function AnimatedRoutes() {
  const location = useLocation()

  return (
    <AnimatePresence mode="wait">
      <Routes location={location} key={location.pathname}>
        <Route
          path="/login"
          element={
            <Page>
              <LoginScreen />
            </Page>
          }
        />
        <Route
          path="/"
          element={
            <RequireAuth>
              <Page>
                <ConnectScreen />
              </Page>
            </RequireAuth>
          }
        />
        <Route
          path="/broadcast"
          element={
            <RequireAdmin>
              <Page>
                <BroadcastScreen />
              </Page>
            </RequireAdmin>
          }
        />
      </Routes>
    </AnimatePresence>
  )
}

function App() {
  return (
    <ToastProvider>
      <HashRouter>
        <AnimatedRoutes />
      </HashRouter>
    </ToastProvider>
  )
}

export default App
