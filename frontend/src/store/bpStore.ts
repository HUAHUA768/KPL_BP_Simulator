import { create } from 'zustand'
import type { BPState, Hero, HeroState } from '../types'

// ============================================================================
// BP Store — 全局 BP 状态管理
// ============================================================================

interface BPStore {
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
  setRoomId: (roomId: string) => void
  setHeroList: (heroes: Hero[]) => void
  updateBPState: (state: BPState) => void
  reset: () => void
}

const initialState = {
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

  setRoomId: (roomId: string) => set({ roomId }),

  setHeroList: (heroes: Hero[]) => set({ heroList: heroes }),

  updateBPState: (state: BPState) =>
    set({
      roomId: state.roomId,
      status: state.status as BPStore['status'],
      blueTeamId: state.blueTeamId,
      redTeamId: state.redTeamId,
      currentTurn: state.currentTurn,
      round: state.round,
      action: state.action,
      side: state.side,
      heroes: state.heroes || [],
      blueBanned: state.blueBanned || [],
      redBanned: state.redBanned || [],
      bluePicked: state.bluePicked || [],
      redPicked: state.redPicked || [],
    }),

  reset: () => set(initialState),
}))