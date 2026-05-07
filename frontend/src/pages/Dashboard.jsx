import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  BarChart3, Server, FileText, HardDrive, Activity,
  ArrowRight, RefreshCw, Calendar
} from 'lucide-react'
import { dashboardAPI, appsAPI } from '../utils/api'

export default function Dashboard() {
  const navigate = useNavigate()
  const [stats, setStats] = useState(null)
  const [apps, setApps] = useState([])
  const [loading, setLoading] = useState(true)

  const fetchData = async () => {
    setLoading(true)
    try {
      const [statsRes, appsRes] = await Promise.all([
        dashboardAPI.getStats(),
        appsAPI.getList(),
      ])
      setStats(statsRes.data)
      setApps(appsRes.data || [])
    } catch (err) {
      // interceptor handles
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
    const timer = setInterval(fetchData, 30000)
    return () => clearInterval(timer)
  }, [])

  const statCards = stats ? [
    {
      label: '注册应用',
      value: stats.total_apps,
      icon: Server,
      color: 'from-blue-500 to-blue-600',
      bgColor: 'bg-blue-50',
      iconColor: 'text-blue-500',
    },
    {
      label: '在线应用',
      value: stats.online_apps,
      icon: Activity,
      color: 'from-emerald-500 to-emerald-600',
      bgColor: 'bg-emerald-50',
      iconColor: 'text-emerald-500',
    },
    {
      label: '日志文件数',
      value: stats.total_log_files,
      icon: FileText,
      color: 'from-amber-500 to-amber-600',
      bgColor: 'bg-amber-50',
      iconColor: 'text-amber-500',
    },
    {
      label: '存储占用',
      value: stats.total_log_size_str || '0 B',
      icon: HardDrive,
      color: 'from-rose-500 to-rose-600',
      bgColor: 'bg-rose-50',
      iconColor: 'text-rose-500',
      isText: true,
    },
  ] : []

  return (
    <div className="animate-fade-in">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">控制面板</h1>
          <p className="text-slate-500 mt-1">实时监控所有应用日志状态</p>
        </div>
        <button
          onClick={fetchData}
          disabled={loading}
          className="flex items-center gap-2 px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-600 hover:bg-slate-50 hover:border-slate-300 transition-all shadow-sm"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {loading && !stats ? (
        <div className="flex items-center justify-center h-64">
          <div className="w-8 h-8 border-3 border-blue-200 border-t-blue-500 rounded-full animate-spin" />
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            {statCards.map((card, i) => (
              <div
                key={i}
                className="bg-white rounded-2xl border border-slate-100 p-5 shadow-sm hover:shadow-md transition-all duration-200"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-sm text-slate-500 mb-1">{card.label}</p>
                    <p className="text-2xl font-bold text-slate-800">
                      {card.isText ? card.value : card.value}
                    </p>
                  </div>
                  <div className={`p-2.5 rounded-xl ${card.bgColor}`}>
                    <card.icon className={`w-5 h-5 ${card.iconColor}`} />
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="mb-6">
            <h2 className="text-lg font-semibold text-slate-800 mb-4">应用列表</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {apps.map((app) => (
                <div
                  key={app.app_id}
                  className="bg-white rounded-2xl border border-slate-100 p-5 shadow-sm hover:shadow-md hover:border-blue-100 transition-all duration-200 cursor-pointer group"
                  onClick={() => navigate(`/logs/${app.app_id}`)}
                >
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <div className="p-2 bg-slate-50 rounded-xl group-hover:bg-blue-50 transition-colors">
                        <Server className="w-5 h-5 text-slate-500 group-hover:text-blue-500 transition-colors" />
                      </div>
                      <div>
                        <h3 className="font-semibold text-slate-800">{app.app_name}</h3>
                        <p className="text-xs text-slate-400 font-mono">{app.app_id}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-1.5">
                      {app.online ? (
                        <>
                          <span className="w-2 h-2 bg-emerald-400 rounded-full animate-pulse-dot" />
                          <span className="text-xs text-emerald-600 font-medium">在线</span>
                        </>
                      ) : (
                        <>
                          <span className="w-2 h-2 bg-slate-300 rounded-full" />
                          <span className="text-xs text-slate-400">离线</span>
                        </>
                      )}
                    </div>
                  </div>

                  <p className="text-sm text-slate-500 mb-4">{app.description}</p>

                  <div className="flex items-center justify-between pt-3 border-t border-slate-50">
                    <div className="flex items-center gap-4 text-xs text-slate-400">
                      <span className="flex items-center gap-1">
                        <FileText className="w-3.5 h-3.5" />
                        {app.total_files} 个文件
                      </span>
                      {app.date_range && app.date_range.length > 0 && (
                        <span className="flex items-center gap-1">
                          <Calendar className="w-3.5 h-3.5" />
                          {app.date_range.length} 天
                        </span>
                      )}
                    </div>
                    <ArrowRight className="w-4 h-4 text-slate-300 group-hover:text-blue-400 group-hover:translate-x-0.5 transition-all" />
                  </div>
                </div>
              ))}
            </div>
          </div>

          {apps.length === 0 && !loading && (
            <div className="text-center py-16 bg-white rounded-2xl border border-slate-100">
              <BarChart3 className="w-12 h-12 text-slate-300 mx-auto mb-3" />
              <p className="text-slate-500">暂无注册应用</p>
              <p className="text-sm text-slate-400 mt-1">请在配置文件中添加应用</p>
            </div>
          )}
        </>
      )}
    </div>
  )
}
