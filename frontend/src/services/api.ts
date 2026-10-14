import axios from 'axios'
import type { APIResponse, Hero, Team, Game, Room, BPState, MatchRecord, BoFormat } from '../types'

// API 基础配置
const BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

export const api = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 响应拦截器
api.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('[API] 请求失败:', error.message)
    return Promise.reject(error)
  }
)

// ============================================================================
// 英雄 API
// ============================================================================

export async function getHeroes(): Promise<Hero[]> {
  const res = await api.get<APIResponse<Hero[]>>('/heroes')
  return res.data.data
}

// ============================================================================
// 房间 API
// ============================================================================

export async function createRoom(
  blueTeamId: number,
  redTeamId: number,
  boFormat: BoFormat = 5,
): Promise<BPState> {
  const res = await api.post<APIResponse<BPState>>('/rooms', {
    blueTeamId,
    redTeamId,
    boFormat,
  })
  return res.data.data
}

export async function joinRoom(roomId: string, teamId: number, side: string): Promise<Room> {
  const res = await api.post<APIResponse<Room>>(`/rooms/${roomId}/join`, {
    teamId,
    side,
  })
  return res.data.data
}

export async function getRoomStatus(roomId: string): Promise<BPState> {
  const res = await api.get<APIResponse<BPState>>(`/rooms/${roomId}/status`)
  return res.data.data
}

export async function banHero(roomId: string, heroId: number): Promise<BPState> {
  const res = await api.post<APIResponse<BPState>>(`/rooms/${roomId}/ban`, {
    heroId,
  })
  return res.data.data
}

export async function pickHero(roomId: string, heroId: number): Promise<BPState> {
  const res = await api.post<APIResponse<BPState>>(`/rooms/${roomId}/pick`, {
    heroId,
  })
  return res.data.data
}

/** 提交本局胜方（BP 完成后必须提交才能进入下一局） */
export async function submitGameResult(
  roomId: string,
  winner: 'blue' | 'red',
): Promise<BPState> {
  const res = await api.post<APIResponse<BPState>>(`/rooms/${roomId}/result`, { winner })
  return res.data.data
}

/** 进入下一局（后端据此应用全局BP记忆规则）
 *
 * 选边（问题4）：可选传入下一局的红蓝归属。不传时沿用当前局（向后兼容）。
 * 单机模式下由用户在局间面板决定谁蓝谁红。
 */
export async function nextGame(
  roomId: string,
  sides?: { blueTeamId: number; redTeamId: number },
): Promise<BPState> {
  const res = await api.post<APIResponse<BPState>>(
    `/rooms/${roomId}/next-game`,
    sides ? { blueTeamId: sides.blueTeamId, redTeamId: sides.redTeamId } : {},
  )
  return res.data.data
}

/** 手动结束比赛（无限制赛制专用） */
export async function finishMatch(roomId: string): Promise<BPState> {
  const res = await api.post<APIResponse<BPState>>(`/rooms/${roomId}/finish-match`, {})
  return res.data.data
}

// ============================================================================
// 对局 / 系列赛 API
// ============================================================================

export async function getGames(matchId?: number): Promise<Game[]> {
  const params = matchId ? { matchId } : {}
  const res = await api.get<APIResponse<Game[]>>('/games', { params })
  return res.data.data
}

/** 系列赛历史（含每局 BP 明细） */
export async function getMatches(): Promise<MatchRecord[]> {
  const res = await api.get<APIResponse<MatchRecord[]>>('/matches')
  return res.data.data
}

/**
 * 从未结束的对局记录恢复房间，继续模拟 BP。
 *
 * 后端优先复用内存中的房间；若后端已重启（房间丢失），
 * 则依据 game / ban_pick_action 回放重建，恢复到"当前进度"。
 * 返回可直接跳转的 roomId。
 */
export async function resumeMatch(matchId: number): Promise<string> {
  const res = await api.post<APIResponse<{ roomId: string; state: BPState }>>(
    `/matches/${matchId}/resume`,
    {},
  )
  return res.data.data.roomId
}

// ============================================================================
// 战队 API
// ============================================================================

/** 战队列表（走真实数据库，含 SYG/WST 等全部队伍） */
export async function getTeams(): Promise<Team[]> {
  const res = await api.get<APIResponse<Team[]>>('/teams')
  return res.data.data
}
