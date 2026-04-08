import { Routes, Route, Navigate } from 'react-router'
import { AppLayout } from './components/layout/AppLayout'
import { LoginPage } from './pages/LoginPage'
import { InboxPage } from './pages/InboxPage'
import { TodayPage } from './pages/TodayPage'
import { UpcomingPage } from './pages/UpcomingPage'
import { ListPage } from './pages/ListPage'
import { LabelPage } from './pages/LabelPage'
import { CompletedPage } from './pages/CompletedPage'
import { TrashPage } from './pages/TrashPage'
import { EisenhowerPage } from './pages/EisenhowerPage'
import { CalendarPage } from './pages/CalendarPage'
import { useTheme, useApplyTheme } from './hooks/useTheme'

function App() {
  // Apply persisted theme at the root so both /login and the main
  // app routes respect the user's preference. AppLayout also calls
  // useApplyTheme — that's a no-op when the value matches.
  const { theme } = useTheme()
  useApplyTheme(theme)

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/inbox" replace />} />
        <Route path="inbox" element={<InboxPage />} />
        <Route path="today" element={<TodayPage />} />
        <Route path="upcoming" element={<UpcomingPage />} />
        <Route path="matrix" element={<EisenhowerPage />} />
        <Route path="calendar" element={<CalendarPage />} />
        <Route path="lists/:id" element={<ListPage />} />
        <Route path="labels/:id" element={<LabelPage />} />
        <Route path="completed" element={<CompletedPage />} />
        <Route path="trash" element={<TrashPage />} />
        <Route path="*" element={<Navigate to="/inbox" replace />} />
      </Route>
    </Routes>
  )
}

export default App
