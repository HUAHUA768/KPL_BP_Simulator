import React from 'react'
import type { Game } from '../types'

interface GameResultTableProps {
  games: Game[]
  getHeroName?: (id: number) => string
}

const GameResultTable: React.FC<GameResultTableProps> = ({ games, getHeroName }) => {
  const heroName = (id: number) => (getHeroName ? getHeroName(id) : `英雄${id}`)

  return (
    <div className="w-full overflow-x-auto">
      <table className="w-full text-sm text-left text-gray-300">
        <thead className="text-xs text-gray-400 uppercase bg-gray-800/50">
          <tr>
            <th className="px-4 py-3">局次</th>
            <th className="px-4 py-3">蓝方</th>
            <th className="px-4 py-3">红方</th>
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
                <td className="px-4 py-3 font-medium text-white">
                  第{game.gameNumber}局
                </td>
                <td className="px-4 py-3">🔵 队伍{game.blueTeamId}</td>
                <td className="px-4 py-3">🔴 队伍{game.redTeamId}</td>
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
              <td colSpan={8} className="px-4 py-8 text-center text-gray-500">
                暂无对局记录
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}

export default GameResultTable