import { useEffect } from 'react'
import { connectWS } from '../services/ws'
import { useBPStore } from '../store/bpStore'
import type { WSMessage, BPState } from '../types'

// useWebSocket — WebSocket 连接管理 Hook
// 通道定位：纯下行状态推送。写路径唯一走 REST；WS 快照由 store 幂等合并。
// 单机模式无需上行能力，统一以 spectator 身份订阅（后端对未知 side 已兜底）。
export function useWebSocket(roomId: string) {
  const updateBPState = useBPStore((s) => s.updateBPState)

  useEffect(() => {
    if (!roomId) return

    const ws = connectWS(roomId, 'spectator', (msg: WSMessage) => {
      switch (msg.type) {
        case 'bp_update':
        case 'bp_finished':
          if (msg.data) {
            updateBPState(msg.data as BPState)
          }
          break
        case 'error':
          console.error('[WS] 错误:', msg.data)
          break
      }
    })

    return () => {
      ws.close()
    }
  }, [roomId, updateBPState])
}
