import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import AppLayout from './components/layout/AppLayout'
import Dashboard from './pages/Dashboard'
import Pipelines from './pages/Pipelines'
import PipelineEditor from './pages/PipelineEditor'
import Settings from './pages/Settings'
import LLMChat from './pages/LLMChat'
import ChannelConnect from './pages/ChannelConnect'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/channels/connect" element={<ChannelConnect />} />
          <Route element={<AppLayout />}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/pipelines" element={<Pipelines />} />
            <Route path="/pipelines/:id" element={<PipelineEditor />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/chat" element={<LLMChat />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
