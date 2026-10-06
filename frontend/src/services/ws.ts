import type { WSMessage } from '../types'

// WebSocket 服务 — 连接 BP 房间并处理实时消息

const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL || `ws://${window.location.host}/ws`

export function connectWS(
  roomId: string,
  side: string,
  onMessage: (msg: WSMessage) => void
): WebSocket {
  const url = `${WS_BASE_URL}?roomId=${encodeURIComponent(roomId)}&side=${encodeURIComponent(side)}`
  const ws = new WebSocket(url)

  ws.onopen = () => {
    console.log('[WS] 已连接:', roomId)
  }

  ws.onmessage = (event) => {
    try {
      const msg: WSMessage = JSON.parse(event.data)
      onMessage(msg)
    } catch (err) {
      console.error('[WS] 消息解析失败:', err)
    }
  }

  ws.onerror = (error) => {
    console.error('[WS] 连接错误:', error)
  }

  ws.onclose = (event) => {
    console.log('[WS] 连接关闭:', event.code, event.reason)
  }

  return ws
}

// 注：上行 ban/pick 协议保留（见 docs/API.md「WebSocket 协议」），
// 但单机模式前端不再发送——写路径唯一走 REST，故此处不再提供发送 API。