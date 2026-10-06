import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { MatchCard } from '../components/GameResultTable'
import { getMatches, getHeroes, getTeams } from '../services/api'
import type { MatchRecord, Hero, Team } from '../types'

const HistoryPage: React.FC = () => {
  const navigate = useNavigate()
  const [matches, setMatches] = useState<MatchRecord[]>([])
  const [heroes, setHeroes] = useState<Hero[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([getMatches(), getHeroes(), getTeams().catch(() => [] as Team[])])
      .then(([matchData, heroData, teamData]) => {
        setMatches(matchData)
        setHeroes(heroData)
        setTeams(teamData)
        setLoading(false)
      })
      .catch((err) => {
        setError('加载失败: ' + (err?.response?.data?.message || err.message))
        setLoading(false)
      })
  }, [])

  const getHeroName = (id: number): string =>
    heroes.find((h) => h.id === id)?.name || `英雄${id}`

  const getTeamName = (id: number): string =>
    teams.find((t) => t.id === id)?.name || `队伍${id}`

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-white text-xl">加载中...</div>
      </div>
    )
  }

  // 只展示有内容的系列赛（避免空壳卡片）
  const visibleMatches = matches.filter((m) => m.games.length > 0 || m.status === 'finished')

  return (
    <div className="min-h-screen bg-gray-900 p-4">
      <div className="max-w-7xl mx-auto">
        {/* 顶部导航（居中） */}
        <div className="relative mb-6 flex items-center justify-center">
          <button
            className="absolute left-0 rounded-md border border-gray-700 bg-gray-800/60 px-3 py-1.5 text-sm text-gray-300 transition-colors hover:border-gray-500 hover:text-white"
            onClick={() => navigate('/')}
          >
            ← 返回首页
          </button>
          <h1 className="text-center text-2xl font-bold text-white">📋 历史对局</h1>
        </div>

        {/* 错误提示 */}
        {error && (
          <div className="bg-red-900/40 border border-red-500 text-red-300 rounded px-4 py-2 mb-4 text-center">
            {error}
          </div>
        )}

        {/* 系列赛卡片列表 */}
        {visibleMatches.length === 0 ? (
          <div className="bg-gray-800/50 rounded-lg border border-gray-700 px-4 py-12 text-center text-gray-500">
            暂无对局记录
          </div>
        ) : (
          visibleMatches.map((match) => (
            <MatchCard
              key={match.id}
              match={match}
              getTeamName={getTeamName}
              getHeroName={getHeroName}
            />
          ))
        )}
      </div>
    </div>
  )
}

export default HistoryPage