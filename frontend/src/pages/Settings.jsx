import { useState, useEffect } from 'react'
import { Shield, Server, Key, RefreshCw } from 'lucide-react'
import { configAPI } from '../utils/api'

export default function Settings() {
  const [config, setConfig] = useState(null)
  const [loading, setLoading] = useState(true)

  const fetchConfig = async () => {
    setLoading(true)
    try {
      const res = await configAPI.get()
      setConfig(res.data)
    } catch (err) {
      // interceptor handles
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchConfig() }, [])

  return (
    <div className="animate-fade-in max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">系统配置</h1>
          <p className="text-slate-500 mt-1">查看当前系统配置信息</p>
        </div>
        <button
          onClick={fetchConfig}
          disabled={loading}
          className="flex items-center gap-2 px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-600 hover:bg-slate-50 hover:border-slate-300 transition-all shadow-sm"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {loading && !config ? (
        <div className="flex items-center justify-center h-64">
          <div className="w-8 h-8 border-3 border-blue-200 border-t-blue-500 rounded-full animate-spin" />
        </div>
      ) : config ? (
        <div className="space-y-6">
          {/* Server Info */}
          <div className="bg-white rounded-2xl border border-slate-100 shadow-sm overflow-hidden">
            <div className="px-6 py-4 border-b border-slate-100 flex items-center gap-3">
              <div className="p-2 bg-blue-50 rounded-xl">
                <Shield className="w-5 h-5 text-blue-500" />
              </div>
              <div>
                <h3 className="font-semibold text-slate-800">服务器配置</h3>
                <p className="text-sm text-slate-500">WSS 安全连接配置</p>
              </div>
            </div>
            <div className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="p-4 bg-slate-50 rounded-xl">
                  <p className="text-xs text-slate-500 mb-1">WSS 端口</p>
                  <p className="text-lg font-semibold text-slate-800 font-mono">{config.wss_port}</p>
                </div>
                <div className="p-4 bg-slate-50 rounded-xl">
                  <p className="text-xs text-slate-500 mb-1">日志保留天数</p>
                  <p className="text-lg font-semibold text-slate-800">{config.max_retain_days} 天</p>
                </div>
              </div>
            </div>
          </div>

          {/* Connection Guide */}
          <div className="bg-white rounded-2xl border border-slate-100 shadow-sm overflow-hidden">
            <div className="px-6 py-4 border-b border-slate-100 flex items-center gap-3">
              <div className="p-2 bg-emerald-50 rounded-xl">
                <Key className="w-5 h-5 text-emerald-500" />
              </div>
              <div>
                <h3 className="font-semibold text-slate-800">接入指南</h3>
                <p className="text-sm text-slate-500">如何接入日志系统</p>
              </div>
            </div>
            <div className="p-6">
              <div className="bg-slate-900 rounded-xl p-4 overflow-auto">
                <pre className="text-sm text-slate-200 font-mono leading-relaxed">
{`// WebSocket 接入示例 (JavaScript)
const ws = new WebSocket('wss://your-host:${config.wss_port}/ws/producer?app_id=your-app-id');

ws.onopen = () => {
  // 发送日志
  ws.send(JSON.stringify({
    type: 'log',
    payload: {
      level: 'INFO',
      message: '应用启动成功',
      source: 'main',
      extra: { version: '1.0.0' }
    }
  }));
};

// 批量发送
ws.send(JSON.stringify({
  type: 'batch_log',
  payload: [
    { level: 'INFO', message: '日志1' },
    { level: 'WARN', message: '日志2' }
  ]
}));`}
                </pre>
              </div>

              <div className="mt-4 bg-slate-900 rounded-xl p-4 overflow-auto">
                <p className="text-xs text-slate-400 mb-2 font-mono"># 证书生成命令</p>
                <pre className="text-sm text-emerald-400 font-mono">
{`./certgen -out ./certs -org "MyOrg" -cn "LogServer" -days 365 -hosts "localhost,192.168.1.100"`}
                </pre>
              </div>
            </div>
          </div>

          {/* App List */}
          <div className="bg-white rounded-2xl border border-slate-100 shadow-sm overflow-hidden">
            <div className="px-6 py-4 border-b border-slate-100 flex items-center gap-3">
              <div className="p-2 bg-amber-50 rounded-xl">
                <Server className="w-5 h-5 text-amber-500" />
              </div>
              <div>
                <h3 className="font-semibold text-slate-800">已注册应用</h3>
                <p className="text-sm text-slate-500">配置文件中定义的允许接入应用</p>
              </div>
            </div>
            <div className="divide-y divide-slate-50">
              {(config.apps || []).map((app, i) => (
                <div key={app.id} className="px-6 py-4 flex items-center justify-between hover:bg-slate-50 transition-colors">
                  <div className="flex items-center gap-3">
                    <span className="w-8 h-8 flex items-center justify-center bg-slate-100 rounded-lg text-sm font-medium text-slate-500">
                      {i + 1}
                    </span>
                    <div>
                      <p className="font-medium text-slate-700">{app.name}</p>
                      <p className="text-xs text-slate-400 font-mono">{app.id}</p>
                    </div>
                  </div>
                  <span className="text-sm text-slate-500">{app.description}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
