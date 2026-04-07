import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'
import ToastContainer from '../ui/Toast'

export default function AppLayout() {
  return (
    <div className="flex h-screen bg-bg text-text overflow-hidden">
      <Sidebar />
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
      <ToastContainer />
    </div>
  )
}
