import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { LogIn, Terminal, Shield, Eye, EyeOff } from 'lucide-react'
import { authAPI } from '../utils/api'
import { showToast } from '../components/Toast'

export default function Login() {
  const navigate = useNavigate()
  const [form, setForm] = useState({ username: '', password: '' })
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()

    if (!form.username.trim()) {
      showToast('请输入用户名', 'warning')
      return
    }
    if (!form.password.trim()) {
      showToast('请输入密码', 'warning')
      return
    }

    setLoading(true)
    try {
      const res = await authAPI.login(form)
      localStorage.setItem('loghub_token', res.data.token)
      localStorage.setItem('loghub_username', res.data.username)
      showToast('登录成功', 'success')
      navigate('/', { replace: true })
    } catch (err) {
      // interceptor handles error display
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 px-4">
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-blue-100 rounded-full opacity-40 blur-3xl" />
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-indigo-100 rounded-full opacity-40 blur-3xl" />
      </div>

      <div className="relative w-full max-w-md">
        <div className="bg-white/80 backdrop-blur-xl rounded-3xl shadow-xl border border-white/60 p-8">
          <div className="text-center mb-8">
            <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-2xl shadow-lg shadow-blue-200 mb-4">
              <Terminal className="w-8 h-8 text-white" />
            </div>
            <h1 className="text-2xl font-bold text-slate-800">LogHub</h1>
            <p className="text-slate-500 mt-1">应用日志集中管理平台</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">用户名</label>
              <div className="relative">
                <input
                  type="text"
                  value={form.username}
                  onChange={(e) => setForm(f => ({ ...f, username: e.target.value }))}
                  className="w-full px-4 py-3 rounded-xl border border-slate-200 bg-slate-50/50 text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all"
                  placeholder="请输入用户名"
                  autoComplete="username"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">密码</label>
              <div className="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={form.password}
                  onChange={(e) => setForm(f => ({ ...f, password: e.target.value }))}
                  className="w-full px-4 py-3 pr-12 rounded-xl border border-slate-200 bg-slate-50/50 text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all"
                  placeholder="请输入密码"
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-slate-400 hover:text-slate-600 transition-colors"
                >
                  {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                </button>
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 px-4 bg-gradient-to-r from-blue-500 to-indigo-600 text-white font-medium rounded-xl shadow-lg shadow-blue-200 hover:shadow-xl hover:shadow-blue-300 hover:from-blue-600 hover:to-indigo-700 focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-all duration-200 disabled:opacity-60 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {loading ? (
                <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              ) : (
                <>
                  <LogIn className="w-5 h-5" />
                  登录
                </>
              )}
            </button>
          </form>

          <div className="mt-6 flex items-center gap-2 px-4 py-3 bg-slate-50 rounded-xl">
            <Shield className="w-4 h-4 text-slate-400 shrink-0" />
            <p className="text-xs text-slate-500">通过 WSS 安全连接传输日志数据，支持自定义 TLS 证书</p>
          </div>
        </div>
      </div>
    </div>
  )
}
