import axios from 'axios'
import type { APIResponse, Hero, Team, Game, Room, BPState } from '../types'

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

export async function createRoom(blueTeamId: number, redTeamId: number): Promise<Room> {
  const res = await api.post<APIResponse<Room>>('/rooms', {
    blueTeamId,
    redTeamId,
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

// ============================================================================
// 对局 API
// ============================================================================

export async function getGames(matchId?: number): Promise<Game[]> {
  const params = matchId ? { matchId } : {}
  const res = await api.get<APIResponse<Game[]>>('/games', { params })
  return res.data.data
}

// ============================================================================
// 战队 API（通过 heroes 接口间接获取，如有需要可扩展）
// ============================================================================

export async function getTeams(): Promise<Team[]> {
  // 目前后端没有独立的 /api/teams 端点，返回预设列表
  // 实际项目里应添加对应端点
  return [
    { id: 1, name: 'AG超玩会', logoUrl: '/images/teams/ag.png' },
    { id: 2, name: '重庆狼队', logoUrl: '/images/teams/wolves.png' },
    { id: 3, name: '武汉eStarPro', logoUrl: '/images/teams/estar.png' },
    { id: 4, name: '广州TTG', logoUrl: '/images/teams/ttg.png' },
    { id: 5, name: '深圳DYG', logoUrl: '/images/teams/dyg.png' },
    { id: 6, name: '济南RW侠', logoUrl: '/images/teams/rw.png' },
  ]
}