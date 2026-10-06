import React from 'react'
import type { Game, MatchRecord } from '../types'

interface GameResultTableProps {
  games: Game[]
  getHeroName?: (id: number) => string
  /** 是否显示蓝方/红方队名列（系列赛卡片内可省略，避免队名重复） */
  showTeams?: boolean
}

const GameResultTable: React.FC<GameResultTableProps> = ({
  games,
  getHeroName,
  showTeams = true,
}) => {
  const heroName = (id: number) => (getHeroName ? getHeroName(id) : `英雄${id}`)

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

            return (
              <tr
                key={game.id}
                className="border-b border-gray-700 hover:bg-gray-800/30 transition-colors"
              >
                <td className="px-4 py-3 font-medium text-white">第{game.gameNumber}局</td>
                {showTeams && <td className="px-4 py-3">🔵 队伍{game.blueTeamId}</td>}
                {showTeams && <td className="px-4 py-3">🔴 队伍{game.redTeamId}</td>}
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
                    <span
                      className={
                        game.winner === 'blue'
                          ? 'text-blue-400 font-bold'
                          : 'text-red-400 font-bold'
                      }
                    >
                      {game.winner === 'blue' ? '🔵 蓝方胜' : '🔴 红方胜'}
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
}

export const MatchCard: React.FC<MatchCardProps> = ({ match, getTeamName, getHeroName }) => {
  const boLabel = match.boFormat > 0 ? `BO${match.boFormat}` : '无限制'
  const statusLabel = match.status === 'finished' ? '已结束' : '进行中'
  const statusClass = match.status === 'finished' ? 'text-gray-400' : 'text-green-400'

  return (
    <div className="bg-gray-800/50 rounded-lg border border-gray-700 overflow-hidden mb-5">
      {/* 头部：队名 + 大比分 + 赛制 + 状态 */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-700 bg-gray-800/70 px-4 py-3">
        <div className="text-base font-bold text-white">
          <span className="text-blue-400">🔵 {getTeamName(match.teamAId)}</span>
          <span className="mx-3 text-lg text-yellow-400">
            {match.teamAScore} : {match.teamBScore}
          </span>
          <span className="text-red-400">{getTeamName(match.teamBId)} 🔴</span>
        </div>
        <div className="text-sm text-gray-400">
          <span className="rounded bg-gray-700/70 px-2 py-0.5 text-gray-200">{boLabel}</span>
          <span className={`ml-3 font-medium ${statusClass}`}>{statusLabel}</span>
          <span className="ml-3">共 {match.games.length} 局</span>
        </div>
      </div>

      <GameResultTable games={match.games} getHeroName={getHeroName} showTeams={false} />
    </div>
  )
}

export default GameResultTable