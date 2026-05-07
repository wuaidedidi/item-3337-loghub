import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  ArrowLeft, Search, Filter, FileText, Calendar, Download,
  ChevronLeft, ChevronRight, Radio, Pause, Play, Trash2,
  AlertCircle, Info, AlertTriangle, Bug, Skull
} from 'lucide-react'
import { logsAPI, appsAPI } from '../utils/api'
import logWS from '../utils/websocket'
import { showToast } from '../components/Toast'

const LEVEL_CONFIG = {
  DEBUG: { icon: Bug, color: 'text-slate-500', bg: 'bg-slate-100', border: 'border-slate-200', label: 'DEBUG' },
  INFO: { icon: Info, color: 'text-blue-600', bg: 'bg-blue-50', border: 'border-blue-200', label: 'INFO' },
  WARN: { icon: AlertTriangle, color: 'text-amber-600', bg: 'bg-amber-50', border: 'border-amber-200', label: 'WARN' },
  ERROR: { icon: AlertCircle, color: 'text-red-600', bg: 'bg-red-50', border: 'border-red-200', label: 'ERROR' },
  FATAL: { icon: Skull, color: 'text-red-800', bg: 'bg-red-100', border: 'border-red-300', label: 'FATAL' },
}

const LEVELS = ['ALL', 'DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL']

export default function LogViewer() {
  const { appId } = useParams()
  const navigate = useNavigate()
  const logEndRef = useRef(null)
  const logContainerRef = useRef(null)

  const [app, setApp] = useState(null)
  const [files, setFiles] = useState([])
  const [selectedDate, setSelectedDate] = useState('')
  const [logs, setLogs] = useState(null)
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState('ALL')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(100)

  // Real-time log state
  const [activeTab, setActiveTab] = useState('history')
  const [realtimeLogs, setRealtimeLogs] = useState([])
  const [wsConnected, setWsConnected] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const [realtimePaused, setRealtimePaused] = useState(false)

  // Fetch app info and log files
  useEffect(() => {
    const fetchAppInfo = async () => {
      try {
        const res = await appsAPI.getList()
        const apps = res.data || []
        const found = apps.find(a => a.app_id === appId)
        if (found) setApp(found)

        const filesRes = await logsAPI.getFiles(appId)
        const fileList = filesRes.data || []
        setFiles(fileList)
        if (fileList.length > 0 && !selectedDate) {
          setSelectedDate(fileList[0].date)
        }
      } catch (err) {
        // interceptor handles
      }
    }
    fetchAppInfo()
  }, [appId])

  // Fetch logs when date/filters change
  useEffect(() => {
    if (!selectedDate || activeTab !== 'history') return
    fetchLogs()
  }, [selectedDate, level, page, activeTab])

  const fetchLogs = async () => {
    if (!selectedDate) return
    setLoading(true)
    try {
      const res = await logsAPI.query({
        app_id: appId,
        date: selectedDate,
        keyword: keyword,
        level: level === 'ALL' ? '' : level,
        page: page,
        page_size: pageSize,
      })
      setLogs(res.data)
    } catch (err) {
      // interceptor handles
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = () => {
    setPage(1)
    fetchLogs()
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') handleSearch()
  }

  // WebSocket for real-time logs
  useEffect(() => {
    if (activeTab !== 'realtime') return

    const statusUnsub = logWS.on('status', (status) => {
      setWsConnected(status.connected)
    })

    const logUnsub = logWS.on('log', (entry) => {
      if (realtimePaused) return
      setRealtimeLogs(prev => {
        const next = [...prev, { ...entry, _id: Date.now() + Math.random() }]
        if (next.length > 500) return next.slice(-500)
        return next
      })
    })

    logWS.connect(appId)

    return () => {
      statusUnsub()
      logUnsub()
      logWS.disconnect()
    }
  }, [activeTab, appId, realtimePaused])

  // Auto-scroll for real-time logs
  useEffect(() => {
    if (autoScroll && activeTab === 'realtime' && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [realtimeLogs, autoScroll, activeTab])

  const clearRealtimeLogs = () => {
    setRealtimeLogs([])
    showToast('实时日志已清空', 'success')
  }

  const formatFileSize = (bytes) => {
    if (bytes >= 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
    if (bytes >= 1024) return (bytes / 1024).toFixed(2) + ' KB'
    return bytes + ' B'
  }

  const getLogLineClass = (levelStr) => {
    const l = (levelStr || 'INFO').toUpperCase()
    return `log-line log-line-${l.toLowerCase()}`
  }

  const getLevelBadge = (levelStr) => {
    const l = (levelStr || 'INFO').toUpperCase()
    const config = LEVEL_CONFIG[l] || LEVEL_CONFIG.INFO
    return (
      <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium ${config.bg} ${config.color} ${config.border} border`}>
        {config.label}
      </span>
    )
  }

  return (
    <div className="animate-fade-in h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between mb-4 shrink-0">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/')}
            className="p-2 rounded-xl bg-white border border-slate-200 text-slate-500 hover:text-slate-700 hover:bg-slate-50 transition-all shadow-sm"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div>
            <h1 className="text-xl font-bold text-slate-800">
              {app?.app_name || appId}
            </h1>
            <p className="text-sm text-slate-400 font-mono">{appId}</p>
          </div>
        </div>
      </div>

      {/* Tab Switcher */}
      <div className="flex items-center gap-1 p-1 bg-slate-100 rounded-xl mb-4 shrink-0 w-fit">
        <button
          onClick={() => setActiveTab('history')}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
            activeTab === 'history'
              ? 'bg-white text-slate-800 shadow-sm'
              : 'text-slate-500 hover:text-slate-700'
          }`}
        >
          <FileText className="w-4 h-4" />
          历史日志
        </button>
        <button
          onClick={() => setActiveTab('realtime')}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
            activeTab === 'realtime'
              ? 'bg-white text-slate-800 shadow-sm'
              : 'text-slate-500 hover:text-slate-700'
          }`}
        >
          <Radio className="w-4 h-4" />
          实时日志
          {wsConnected && activeTab === 'realtime' && (
            <span className="w-2 h-2 bg-emerald-400 rounded-full animate-pulse-dot" />
          )}
        </button>
      </div>

      {/* History Tab */}
      {activeTab === 'history' && (
        <div className="flex gap-4 flex-1 min-h-0">
          {/* Sidebar - File List */}
          <div className="w-64 shrink-0 bg-white rounded-2xl border border-slate-100 shadow-sm flex flex-col">
            <div className="px-4 py-3 border-b border-slate-100">
              <h3 className="text-sm font-semibold text-slate-700 flex items-center gap-2">
                <Calendar className="w-4 h-4" />
                日志文件
              </h3>
            </div>
            <div className="flex-1 overflow-y-auto p-2">
              {files.length === 0 ? (
                <div className="text-center py-8">
                  <FileText className="w-8 h-8 text-slate-300 mx-auto mb-2" />
                  <p className="text-sm text-slate-400">暂无日志文件</p>
                </div>
              ) : (
                files.map((file) => (
                  <button
                    key={file.date}
                    onClick={() => { setSelectedDate(file.date); setPage(1) }}
                    className={`w-full text-left px-3 py-2.5 rounded-xl mb-1 transition-all ${
                      selectedDate === file.date
                        ? 'bg-blue-50 border border-blue-100 text-blue-700'
                        : 'hover:bg-slate-50 text-slate-600'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">{file.date}</span>
                    </div>
                    <span className="text-xs text-slate-400 mt-0.5 block">
                      {formatFileSize(file.size)}
                    </span>
                  </button>
                ))
              )}
            </div>
          </div>

          {/* Main Content - Log Lines */}
          <div className="flex-1 flex flex-col min-w-0 bg-white rounded-2xl border border-slate-100 shadow-sm">
            {/* Filters */}
            <div className="px-4 py-3 border-b border-slate-100 flex items-center gap-3 flex-wrap">
              <div className="relative flex-1 min-w-[200px]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <input
                  type="text"
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="搜索关键词..."
                  className="w-full pl-10 pr-4 py-2 rounded-xl border border-slate-200 bg-slate-50/50 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all"
                />
              </div>

              <div className="relative">
                <Filter className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <select
                  value={level}
                  onChange={(e) => { setLevel(e.target.value); setPage(1) }}
                  className="pl-10 pr-8 py-2 rounded-xl border border-slate-200 bg-slate-50/50 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all appearance-none cursor-pointer"
                >
                  {LEVELS.map(l => (
                    <option key={l} value={l}>{l === 'ALL' ? '全部级别' : l}</option>
                  ))}
                </select>
              </div>

              <button
                onClick={handleSearch}
                className="px-4 py-2 bg-blue-500 text-white text-sm font-medium rounded-xl hover:bg-blue-600 transition-colors shadow-sm"
              >
                搜索
              </button>
            </div>

            {/* Log Content */}
            <div className="flex-1 overflow-auto p-4 min-h-0" ref={logContainerRef}>
              {loading ? (
                <div className="flex items-center justify-center h-full">
                  <div className="w-6 h-6 border-2 border-blue-200 border-t-blue-500 rounded-full animate-spin" />
                </div>
              ) : !logs || !logs.lines || logs.lines.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-full text-slate-400">
                  <FileText className="w-12 h-12 mb-3 text-slate-300" />
                  <p className="text-sm">
                    {selectedDate ? '该日期无匹配的日志记录' : '请选择一个日志文件'}
                  </p>
                </div>
              ) : (
                <div className="space-y-0.5">
                  {logs.lines.map((line, idx) => (
                    <div
                      key={`${line.line_number}-${idx}`}
                      className={`flex items-start gap-3 px-3 py-1.5 rounded-lg hover:bg-slate-50 transition-colors group ${getLogLineClass(line.level)}`}
                    >
                      <span className="text-xs text-slate-300 font-mono w-10 shrink-0 text-right pt-0.5 select-none">
                        {line.line_number}
                      </span>
                      {getLevelBadge(line.level)}
                      <span className="flex-1 break-all whitespace-pre-wrap text-sm font-mono leading-relaxed">
                        {line.content}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Pagination */}
            {logs && logs.total_pages > 1 && (
              <div className="px-4 py-3 border-t border-slate-100 flex items-center justify-between">
                <span className="text-sm text-slate-500">
                  共 {logs.total} 条记录，第 {logs.page}/{logs.total_pages} 页
                </span>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setPage(p => Math.max(1, p - 1))}
                    disabled={page <= 1}
                    className="p-2 rounded-lg border border-slate-200 text-slate-500 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </button>
                  <span className="text-sm text-slate-600 px-3">{page}</span>
                  <button
                    onClick={() => setPage(p => Math.min(logs.total_pages, p + 1))}
                    disabled={page >= logs.total_pages}
                    className="p-2 rounded-lg border border-slate-200 text-slate-500 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                  >
                    <ChevronRight className="w-4 h-4" />
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Real-time Tab */}
      {activeTab === 'realtime' && (
        <div className="flex-1 flex flex-col min-h-0 bg-white rounded-2xl border border-slate-100 shadow-sm">
          {/* Realtime Controls */}
          <div className="px-4 py-3 border-b border-slate-100 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium ${
                wsConnected ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700'
              }`}>
                <span className={`w-2 h-2 rounded-full ${
                  wsConnected ? 'bg-emerald-400 animate-pulse-dot' : 'bg-red-400'
                }`} />
                {wsConnected ? '已连接' : '未连接'}
              </div>
              <span className="text-sm text-slate-400">
                {realtimeLogs.length} 条实时日志
              </span>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setAutoScroll(!autoScroll)}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  autoScroll
                    ? 'bg-blue-50 text-blue-600 border border-blue-100'
                    : 'bg-slate-50 text-slate-500 border border-slate-200'
                }`}
              >
                <Download className="w-3.5 h-3.5" />
                自动滚动
              </button>
              <button
                onClick={() => setRealtimePaused(!realtimePaused)}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  realtimePaused
                    ? 'bg-amber-50 text-amber-600 border border-amber-100'
                    : 'bg-slate-50 text-slate-500 border border-slate-200'
                }`}
              >
                {realtimePaused ? <Play className="w-3.5 h-3.5" /> : <Pause className="w-3.5 h-3.5" />}
                {realtimePaused ? '继续' : '暂停'}
              </button>
              <button
                onClick={clearRealtimeLogs}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-50 text-slate-500 border border-slate-200 hover:bg-red-50 hover:text-red-500 hover:border-red-100 transition-all"
              >
                <Trash2 className="w-3.5 h-3.5" />
                清空
              </button>
            </div>
          </div>

          {/* Realtime Log Content */}
          <div className="flex-1 overflow-auto p-4 bg-slate-900 rounded-b-2xl min-h-0">
            {realtimeLogs.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-slate-500">
                <Radio className="w-12 h-12 mb-3 text-slate-600" />
                <p className="text-sm">等待实时日志...</p>
                <p className="text-xs text-slate-600 mt-1">
                  当应用通过 WebSocket 推送日志时，将在这里实时显示
                </p>
                <p className="text-xs text-slate-700 mt-3 px-4 py-2 bg-slate-800 rounded-lg border border-slate-700">
                  想测试此功能？请查看项目 README 中「实时日志快速测试」章节
                </p>
              </div>
            ) : (
              <div className="space-y-0.5">
                {realtimeLogs.map((entry, idx) => {
                  const levelStr = (entry.level || 'INFO').toUpperCase()
                  const colorClass = {
                    DEBUG: 'text-slate-400',
                    INFO: 'text-emerald-400',
                    WARN: 'text-amber-400',
                    ERROR: 'text-red-400',
                    FATAL: 'text-red-300',
                  }[levelStr] || 'text-slate-400'

                  return (
                    <div
                      key={entry._id || idx}
                      className="flex items-start gap-2 font-mono text-[13px] leading-relaxed hover:bg-slate-800/50 px-2 py-0.5 rounded transition-colors"
                    >
                      <span className="text-slate-600 shrink-0">
                        {entry.timestamp ? new Date(entry.timestamp).toLocaleTimeString() : '--:--:--'}
                      </span>
                      <span className={`shrink-0 font-semibold w-14 text-right ${colorClass}`}>
                        [{levelStr}]
                      </span>
                      {entry.source && (
                        <span className="text-sky-400 shrink-0">[{entry.source}]</span>
                      )}
                      <span className="text-slate-200 break-all">{entry.message}</span>
                    </div>
                  )
                })}
                <div ref={logEndRef} />
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
