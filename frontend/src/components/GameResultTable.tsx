import React from 'react'
import type { Game, MatchRecord } from '../types'

interface GameResultTableProps {
  games: Game[]
  getHeroName?: (id: number) => string
  getTeamName?: (id: number) => string
  /** 是否显示蓝方/红方队名列（系列赛卡片内可省略，避免队名重复） */
  showTeams?: boolean
}

const GameResultTable: React.FC<GameResultTableProps> = ({
  games,
  getHeroName,
  getTeamName,
  showTeams = true,
}) => {
  const heroName = (id: number) => (getHeroName ? getHeroName(id) : `英雄${id}`)
  // 队伍名缺失时降级为 "队伍{id}"，保证历史记录永远可读
  const teamName = (id: number) =>
    getTeamName ? getTeamName(id) : `队伍${id}`

  return (
    <div className="w-full overflow-x-auto">
      <table className="w-full text-sm text-left text-gray-300">
        <thead className="text-xs text-gray-400 uppercase bg-gray-800/50">
          <tr>
            <th className="px-4 py-3">局次</th>
            {showTeams && <th className="px-4 py-3">蓝方</th>}
            {showTeams && <th className="px-4 py-3">红方</th>}
            <th className="px-4 py-3">蓝方阵容</th>
            <th className="px-4 py-3">红方阵容</th>
            <th className="px-4 py-3">蓝方禁用</th>
            <th className="px-4 py-3">红方禁用</th>
            <th className="px-4 py-3">胜方</th>
          </tr>
        </thead>
        <tbody>
          {games.map((game) => {
            const actions = game.banPickActions || []
            const bluePicks = actions.filter((a) => a.side === 'blue' && a.actionType === 'pick')
            const redPicks = actions.filter((a) => a.side === 'red' && a.actionType === 'pick')
            const blueBans = actions.filter((a) => a.side === 'blue' && a.actionType === 'ban')
            const redBans = actions.filter((a) => a.side === 'red' && a.actionType === 'ban')

            // 问题1：红蓝归属是"每局"概念，第2局起可能换边，
            // 因此必须用**本局**的 blueTeamId / redTeamId 解析真实队名，
            // 不能沿用系列赛头部的 teamA / teamB（它们只是固定身份）。
            const blueTeamName = teamName(game.blueTeamId)
            const redTeamName = teamName(game.redTeamId)

            return (
              <tr
                key={game.id}
                className="border-b border-gray-700 hover:bg-gray-800/30 transition-colors"
              >
                <td className="px-4 py-3 font-medium text-white">第{game.gameNumber}局</td>
                {showTeams && <td className="px-4 py-3 text-blue-300">🔵 {blueTeamName}</td>}
                {showTeams && <td className="px-4 py-3 text-red-300">🔴 {redTeamName}</td>}
                <td className="px-4 py-3">
                  {bluePicks.map((a) => heroName(a.heroId)).join(', ') || '-'}
                </td>
                <td className="px-4 py-3">
                  {redPicks.map((a) => heroName(a.heroId)).join(', ') || '-'}
                </td>
                <td className="px-4 py-3">
                  {blueBans.map((a) => heroName(a.heroId)).join(', ') || '-'}
                </td>
                <td className="px-4 py-3">
                  {redBans.map((a) => heroName(a.heroId)).join(', ') || '-'}
                </td>
                <td className="px-4 py-3">
                  {game.winner ? (
                    /* 问题1：胜方直接显示真实队名，避免"蓝方胜"在换边后产生歧义 */
                    <span
                      className={
                        game.winner === 'blue'
                          ? 'text-blue-400 font-bold'
                          : 'text-red-400 font-bold'
                      }
                    >
                      {game.winner === 'blue' ? '🔵 ' : '🔴 '}
                      {game.winner === 'blue' ? blueTeamName : redTeamName} 胜
                    </span>
                  ) : (
                    <span className="text-gray-500">-</span>
                  )}
                </td>
              </tr>
            )
          })}
          {games.length === 0 && (
            <tr>
              <td colSpan={showTeams ? 8 : 6} className="px-4 py-8 text-center text-gray-500">
                暂无对局记录
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}

// ============================================================================
// MatchCard — 一场系列赛（BO5/BO7/无限制）一张卡片，内含每局明细
// ============================================================================
interface MatchCardProps {
  match: MatchRecord
  getTeamName: (id: number) => string
  getHeroName: (id: number) => string
  /** 点击「继续模拟 BP」（仅未结束的比赛会展示该按钮） */
  onResume?: (match: MatchRecord) => void
  /** 正在恢复中（禁用按钮防重复点击） */
  resuming?: boolean
}

export const MatchCard: React.FC<MatchCardProps> = ({
  match,
  getTeamName,
  getHeroName,
  onResume,
  resuming = false,
}) => {
  const boLabel = match.boFormat > 0 ? `BO${match.boFormat}` : '无限制'
  const isFinished = match.status === 'finished'
  const statusLabel = isFinished ? '已结束' : '进行中'
  const statusClass = isFinished ? 'text-gray-400' : 'text-green-400'

  // 未结束的比赛才提供"继续模拟 BP"入口
  const canResume = !isFinished && !!onResume

  // 描述当前进度，让用户知道会从哪继续
  const cur = match.games.find((g) => g.gameNumber === match.currentGameNum) ||
    match.games[match.games.length - 1]
  const progressHint = (() => {
    if (!cur) return ''
    const done = (cur.banPickActions || []).length
    if (!cur.bpCompleted) return `第${cur.gameNumber}局 BP 进行中（${done}/20步）`
    if (!cur.winner) return `第${cur.gameNumber}局 BP 已完成，待录入胜方`
    return `第${cur.gameNumber}局已结束`
  })()

  return (
    <div className="bg-gray-800/50 rounded-lg border border-gray-700 overflow-hidden mb-5">
      {/* 头部：队名 + 大比分 + 赛制 + 状态 */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-700 bg-gray-800/70 px-4 py-3">
        {/*
          问题1：大比分是"队伍身份"比分（teamA : teamB），与红蓝归属无关
          （第2局起可能换边）。因此头部**不再**给队名标 🔵/🔴，
          颜色归属由下方每局的明细行按各局实际情况显示，避免误导。
        */}
        <div className="text-base font-bold text-white">
          <span>{getTeamName(match.teamAId)}</span>
          <span className="mx-3 text-lg text-yellow-400">
            {match.teamAScore} : {match.teamBScore}
          </span>
          <span>{getTeamName(match.teamBId)}</span>
          <span className="ml-2 align-middle text-xs font-normal text-gray-400">（总比分）</span>
        </div>
        <div className="flex flex-wrap items-center gap-3 text-sm text-gray-400">
          <span className="rounded bg-gray-700/70 px-2 py-0.5 text-gray-200">{boLabel}</span>
          <span className={`font-medium ${statusClass}`}>{statusLabel}</span>
          <span>共 {match.games.length} 局</span>

          {/* 继续模拟 BP：仅未结束的比赛可见 */}
          {canResume && (
            <button
              type="button"
              disabled={resuming}
              onClick={() => onResume?.(match)}
              title={progressHint}
              className="rounded-md bg-yellow-500 px-3 py-1 text-xs font-bold text-gray-900 transition-colors hover:bg-yellow-400 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {resuming ? '恢复中…' : '▶ 继续模拟 BP'}
            </button>
          )}
        </div>
      </div>

      {/* 进度提示（仅未结束的比赛） */}
      {canResume && progressHint && (
        <div className="border-b border-gray-700/60 bg-gray-900/40 px-4 py-1.5 text-xs text-gray-400">
          点击「继续模拟 BP」将恢复到：{progressHint}
        </div>
      )}

      <GameResultTable
        games={match.games}
        getHeroName={getHeroName}
        getTeamName={getTeamName}
        showTeams
      />
    </div>
  )
}

export default GameResultTable