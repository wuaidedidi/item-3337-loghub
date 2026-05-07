import { useState } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard, Settings, LogOut, Terminal,
  Menu, X, ChevronRight, User
} from 'lucide-react'
import { showToast } from './Toast'

const NAV_ITEMS = [
  { to: '/', icon: LayoutDashboard, label: '控制面板', exact: true },
  { to: '/settings', icon: Settings, label: '系统配置' },
]

export default function Layout() {
  const navigate = useNavigate()
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const username = localStorage.getItem('loghub_username') || 'admin'

  const handleLogout = () => {
    localStorage.removeItem('loghub_token')
    localStorage.removeItem('loghub_username')
    showToast('已退出登录', 'success')
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex h-screen bg-slate-50">
      {/* Sidebar */}
      <aside className={`${sidebarOpen ? 'w-60' : 'w-[72px]'} bg-white border-r border-slate-100 flex flex-col transition-all duration-300 shrink-0`}>
        {/* Logo */}
        <div className="h-16 flex items-center px-4 border-b border-slate-100 gap-3">
          <div className="w-9 h-9 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl flex items-center justify-center shrink-0 shadow-sm">
            <Terminal className="w-5 h-5 text-white" />
          </div>
          {sidebarOpen && (
            <div className="overflow-hidden">
              <h1 className="text-base font-bold text-slate-800 whitespace-nowrap">LogHub</h1>
              <p className="text-[10px] text-slate-400 whitespace-nowrap">日志集中管理</p>
            </div>
          )}
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-3 space-y-1">
          {NAV_ITEMS.map(item => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.exact}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-200 group ${
                  isActive
                    ? 'bg-blue-50 text-blue-600'
                    : 'text-slate-500 hover:bg-slate-50 hover:text-slate-700'
                }`
              }
            >
              <item.icon className="w-5 h-5 shrink-0" />
              {sidebarOpen && <span>{item.label}</span>}
            </NavLink>
          ))}
        </nav>

        {/* User Section */}
        <div className="p-3 border-t border-slate-100">
          <div className={`flex items-center ${sidebarOpen ? 'gap-3 px-3' : 'justify-center'} py-2`}>
            <div className="w-8 h-8 bg-gradient-to-br from-slate-100 to-slate-200 rounded-lg flex items-center justify-center shrink-0">
              <User className="w-4 h-4 text-slate-500" />
            </div>
            {sidebarOpen && (
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-slate-700 truncate">{username}</p>
                <p className="text-xs text-slate-400">管理员</p>
              </div>
            )}
          </div>
          <button
            onClick={handleLogout}
            className={`flex items-center gap-3 w-full px-3 py-2.5 rounded-xl text-sm text-slate-500 hover:bg-red-50 hover:text-red-500 transition-all mt-1 ${!sidebarOpen ? 'justify-center' : ''}`}
          >
            <LogOut className="w-5 h-5 shrink-0" />
            {sidebarOpen && <span>退出登录</span>}
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Top Bar */}
        <header className="h-16 bg-white border-b border-slate-100 flex items-center px-6 shrink-0">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-50 transition-all mr-4"
          >
            {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
          <div className="flex items-center gap-1 text-sm text-slate-400">
            <Terminal className="w-4 h-4" />
            <ChevronRight className="w-3 h-3" />
            <span className="text-slate-600">应用日志中心</span>
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
