class LogWebSocket {
  constructor() {
    this.ws = null
    this.listeners = new Map()
    this.reconnectTimer = null
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 10
    this.appId = 'all'
    this.isConnected = false
  }

  connect(appId = 'all') {
    this.appId = appId
    this.disconnect()

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${protocol}//${host}/ws/viewer?app_id=${appId}`

    try {
      this.ws = new WebSocket(url)

      this.ws.onopen = () => {
        this.isConnected = true
        this.reconnectAttempts = 0
        this.emit('status', { connected: true })
      }

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (data.type === 'log' && data.data) {
            this.emit('log', data.data)
          } else if (data.type === 'connected') {
            this.emit('status', { connected: true, info: data })
          } else if (data.type === 'subscribed') {
            this.emit('subscribed', data)
          }
        } catch (e) {
          // ignore malformed messages
        }
      }

      this.ws.onclose = () => {
        this.isConnected = false
        this.emit('status', { connected: false })
        this.scheduleReconnect()
      }

      this.ws.onerror = () => {
        this.isConnected = false
        this.emit('status', { connected: false })
      }
    } catch (e) {
      this.emit('status', { connected: false })
      this.scheduleReconnect()
    }
  }

  subscribe(appId) {
    this.appId = appId
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'subscribe',
        payload: { app_id: appId }
      }))
    }
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.onclose = null
      this.ws.close()
      this.ws = null
    }
    this.isConnected = false
  }

  scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
    this.reconnectAttempts++

    this.reconnectTimer = setTimeout(() => {
      this.connect(this.appId)
    }, delay)
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event).add(callback)
    return () => this.off(event, callback)
  }

  off(event, callback) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).delete(callback)
    }
  }

  emit(event, data) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).forEach(cb => cb(data))
    }
  }

  destroy() {
    this.disconnect()
    this.listeners.clear()
  }
}

const logWS = new LogWebSocket()
export default logWS
