import { create } from 'zustand'
import type { BPState, Hero, HeroState, TeamBrief } from '../types'
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
  /** 双方队名 + 队标（问题1；缺失时棋盘降级显示红蓝方） */
  blueTeam: TeamBrief | null
  redTeam: TeamBrief | null

  // 系列赛（多局）上下文
  matchId: number | null
  gameNumber: number
  boFormat: number
  blueScore: number
  redScore: number
  winner: 'blue' | 'red' | null
  matchStatus: 'ongoing' | 'finished' | null

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
  blueTeam: null as TeamBrief | null,
  redTeam: null as TeamBrief | null,
  matchId: null,
  gameNumber: 1,
  boFormat: 5,
  blueScore: 0,
  redScore: 0,
  winner: null as 'blue' | 'red' | null,
  matchStatus: null as 'ongoing' | 'finished' | null,
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

  // 幂等入口：REST 响应与 WS 广播都会写入本入口。
  //
  // 版本号 = (gameNumber, currentTurn) 双因子。多局流程下 currentTurn 会从 21
  // 重置为 1，若只比 currentTurn，新一局的快照会被误判为"过期数据"而丢弃，
  // 表现为"点了下一局界面不动"。因此必须先比局号、再比步号。
  updateBPState: (incoming: BPState) =>
    set((state) => {
      // 换房间时旧版本号无意义，直接应用新房间快照
      const sameRoom = !incoming.roomId || !state.roomId || incoming.roomId === state.roomId
      const version = (game: number, turn: number) => (game || 1) * 100 + (turn || 0)
      if (
        sameRoom &&
        version(incoming.gameNumber, incoming.currentTurn) <
          version(state.gameNumber, state.currentTurn)
      ) {
        return state
      }
      return {
        roomId: incoming.roomId,
        status: incoming.status as BPStore['status'],
        blueTeamId: incoming.blueTeamId,
        redTeamId: incoming.redTeamId,
        blueTeam: incoming.blueTeam ?? null,
        redTeam: incoming.redTeam ?? null,
        matchId: incoming.matchId ?? null,
        gameNumber: incoming.gameNumber ?? 1,
        boFormat: incoming.boFormat ?? 5,
        blueScore: incoming.blueScore ?? 0,
        redScore: incoming.redScore ?? 0,
        winner: incoming.winner ?? null,
        matchStatus: (incoming.matchStatus as BPStore['matchStatus']) ?? null,
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