import { create } from 'zustand'
import type { BPState, Hero, HeroState } from '../types'
import type { Mode } from '../utils/actor'

// ============================================================================
// BP Store — 全局 BP 状态管理
// ============================================================================

interface BPStore {
  // 交互模式（solo 一人分饰两角 / dual 双人预留）
  mode: Mode

  // 房间信息
  roomId: string | null
  status: 'waiting' | 'ongoing' | 'finished' | null
  blueTeamId: number | null
  redTeamId: number | null

  // BP 进度
  currentTurn: number
  round: number
  action: string
  side: string

  // 英雄状态
  heroes: HeroState[]
  blueBanned: number[]
  redBanned: number[]
  bluePicked: number[]
  redPicked: number[]

  // 英雄列表（从 API 加载的完整英雄数据）
  heroList: Hero[]

  // 操作
  setMode: (mode: Mode) => void
  setRoomId: (roomId: string) => void
  setHeroList: (heroes: Hero[]) => void
  updateBPState: (state: BPState) => void
  reset: () => void
}

const initialState = {
  mode: 'solo' as Mode,
  roomId: null,
  status: null as 'waiting' | 'ongoing' | 'finished' | null,
  blueTeamId: null,
  redTeamId: null,
  currentTurn: 0,
  round: 0,
  action: '',
  side: '',
  heroes: [] as HeroState[],
  blueBanned: [] as number[],
  redBanned: [] as number[],
  bluePicked: [] as number[],
  redPicked: [] as number[],
  heroList: [] as Hero[],
}

export const useBPStore = create<BPStore>((set) => ({
  ...initialState,

  setMode: (mode: Mode) => set({ mode }),

  setRoomId: (roomId: string) => set({ roomId }),

  setHeroList: (heroes: Hero[]) => set({ heroList: heroes }),

  // 幂等入口：REST 响应与 WS 广播都会写入本入口，以 currentTurn 为版本号。
  // 只应用不落后于本地的快照（相等也应用，REST 响应是权威数据）；
  // 更旧的快照（乱序/重复的 WS 下行）直接丢弃——updater 返回 state 本身时
  // zustand 会因 Object.is 相等而跳过本次更新。
  updateBPState: (incoming: BPState) =>
    set((state) => {
      // 换房间时旧版本号无意义，直接应用新房间快照
      const sameRoom = !incoming.roomId || !state.roomId || incoming.roomId === state.roomId
      if (sameRoom && incoming.currentTurn < state.currentTurn) return state
      return {
        roomId: incoming.roomId,
        status: incoming.status as BPStore['status'],
        blueTeamId: incoming.blueTeamId,
        redTeamId: incoming.redTeamId,
        currentTurn: incoming.currentTurn,
        round: incoming.round,
        action: incoming.action,
        side: incoming.side,
        heroes: incoming.heroes || [],
        blueBanned: incoming.blueBanned || [],
        redBanned: incoming.redBanned || [],
        bluePicked: incoming.bluePicked || [],
        redPicked: incoming.redPicked || [],
      }
    }),

  reset: () => set(initialState),
}))