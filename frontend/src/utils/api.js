import axios from 'axios'

const API_BASE = '/api'

const api = axios.create({
  baseURL: API_BASE,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Recent messages dedup set
const recentMessages = new Set()

function showToast(message, type = 'error') {
  if (recentMessages.has(message)) return
  recentMessages.add(message)
  setTimeout(() => recentMessages.delete(message), 2000)

  window.dispatchEvent(new CustomEvent('loghub-toast', {
    detail: { message, type }
  }))
}

// Request interceptor
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('loghub_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor
api.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 200) {
      showToast(res.message || '请求失败')
      const error = new Error(res.message)
      error._isBusinessError = true
      return Promise.reject(error)
    }
    return res
  },
  (error) => {
    if (error._isBusinessError) {
      return Promise.reject(error)
    }

    if (error.response) {
      const { status, data } = error.response
      let message = data?.message || ''

      switch (status) {
        case 401:
          message = message || '登录已过期，请重新登录'
          localStorage.removeItem('loghub_token')
          localStorage.removeItem('loghub_username')
          if (window.location.pathname !== '/login') {
            window.location.href = '/login'
          }
          break
        case 403:
          message = message || '没有访问权限'
          break
        case 404:
          message = message || '请求的资源不存在'
          break
        case 500:
          message = message || '服务器错误，请稍后重试'
          break
        default:
          message = message || `请求失败 (${status})`
      }

      showToast(message)
    } else if (error.request) {
      showToast('服务器连接失败，请检查网络')
    }

    return Promise.reject(error)
  }
)

export const authAPI = {
  login: (data) => api.post('/login', data),
}

export const dashboardAPI = {
  getStats: () => api.get('/dashboard'),
}

export const appsAPI = {
  getList: () => api.get('/apps'),
}

export const logsAPI = {
  getFiles: (appId) => api.get('/logs/files', { params: { app_id: appId } }),
  query: (params) => api.get('/logs/query', { params }),
}

export const configAPI = {
  get: () => api.get('/config'),
}

export default api
