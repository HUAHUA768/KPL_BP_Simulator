// ============================================================================
// TypeScript 类型定义
// ============================================================================

// 英雄
export interface Hero {
  id: number
  name: string
  iconPath: string
  lanes: string[]
  status?: HeroStatus
  permissions?: HeroPermissions
}

// 英雄状态枚举
export type HeroStatus =
  | 'available'
  | 'tournament_banned'
  | 'banned_by_blue'
  | 'banned_by_red'
  | 'picked_by_blue'
  | 'picked_by_red'
  | 'restricted_by_rule'
  | 'unknown'

// 英雄权限（全局BP记忆）
export interface HeroPermissions {
  blueCanBan: boolean
  blueCanPick: boolean
  redCanBan: boolean
  redCanPick: boolean
}

// 战队
export interface Team {
  id: number
  name: string
  logoUrl: string
}

/** 队伍摘要（随房间状态下发，用于 BP 棋盘展示队名 + 队标） */
export interface TeamBrief {
  id: number
  name: string
  logoUrl: string
}

// 对局
export interface Game {
  id: number
  matchId: number
  gameNumber: number
  blueTeamId: number
  redTeamId: number
  winner: 'blue' | 'red' | null
  bpCompleted: boolean
  banPickActions?: BPAction[]
}

// BP 操作记录
export interface BPAction {
  stepOrder: number
  actionType: 'ban' | 'pick'
  side: 'blue' | 'red'
  heroId: number
}

// BP 状态（从服务端推送）
export interface BPState {
  roomId: string
  status: 'waiting' | 'ongoing' | 'finished'
  // 系列赛（多局）上下文
  matchId: number
  gameNumber: number
  boFormat: number // 5=BO5, 7=BO7, 0=无限制
  blueScore: number
  redScore: number
  /** 本局已提交的胜方；进入下一局后重置为 null */
  winner: 'blue' | 'red' | null
  matchStatus: 'ongoing' | 'finished'
  blueTeamId: number
  redTeamId: number
  /** 双方队名 + 队标（后端 buildRoomState 下发；缺失时降级为红蓝方） */
  blueTeam?: TeamBrief | null
  redTeam?: TeamBrief | null
  currentTurn: number
  round: number
  action: string
  side: string
  blueBanned: number[]
  redBanned: number[]
  bluePicked: number[]
  redPicked: number[]
  heroes: HeroState[]
}

/** 系列赛制选项 */
export type BoFormat = 5 | 7 | 0

/** 一场系列赛（含每局 BP） */
export interface MatchRecord {
  id: number
  teamAId: number
  teamBId: number
  teamAScore: number
  teamBScore: number
  boFormat: number
  status: 'pending' | 'ongoing' | 'finished'
  currentGameNum: number
  createdAt: string
  games: Game[]
}

// 英雄状态（BP 推送中的单个英雄）
export interface HeroState {
  id: number
  status: string
}

// 房间
export interface Room {
  roomId: string
  status: 'waiting' | 'ongoing' | 'finished'
  side?: string
  matchId?: number
  gameNumber?: number
  boFormat?: number
}

// API 响应
export interface APIResponse<T = unknown> {
  code: number
  message?: string
  data: T
}

// WebSocket 消息
export interface WSMessage {
  type: 'bp_update' | 'error' | 'bp_finished'
  data?: BPState | { code: number; message: string }
}