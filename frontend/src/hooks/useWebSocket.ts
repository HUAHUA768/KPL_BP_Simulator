import { useEffect, useRef, useCallback } from 'react'
import { connectWS, sendBan, sendPick } from '../services/ws'
import { useBPStore } from '../store/bpStore'
import type { WSMessage, BPState } from '../types'

// useWebSocket — WebSocket 连接管理 Hook
export function useWebSocket(roomId: string, side: string) {
  const wsRef = useRef<WebSocket | null>(null)
  const updateBPState = useBPStore((s) => s.updateBPState)

  useEffect(() => {
    if (!roomId) return

    const ws = connectWS(roomId, side, (msg: WSMessage) => {
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

    wsRef.current = ws

    return () => {
      ws.close()
      wsRef.current = null
    }
  }, [roomId, side, updateBPState])

  const ban = useCallback((heroId: number) => {
    if (wsRef.current) {
      sendBan(wsRef.current, heroId)
    }
  }, [])

  const pick = useCallback((heroId: number) => {
    if (wsRef.current) {
      sendPick(wsRef.current, heroId)
    }
  }, [])

  return { ban, pick }
}