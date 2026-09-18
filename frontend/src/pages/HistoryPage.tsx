import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import GameResultTable from '../components/GameResultTable'
import { getGames, getHeroes } from '../services/api'
import type { Game, Hero } from '../types'

const HistoryPage: React.FC = () => {
  const navigate = useNavigate()
  const [games, setGames] = useState<Game[]>([])
  const [heroes, setHeroes] = useState<Hero[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([getGames(), getHeroes()])
      .then(([gamesData, heroesData]) => {
        setGames(gamesData)
        setHeroes(heroesData)
        setLoading(false)
      })
      .catch((err) => {
        setError('加载失败: ' + err.message)
        setLoading(false)
      })
  }, [])

  const getHeroName = (id: number): string => {
    return heroes.find((h) => h.id === id)?.name || `英雄${id}`
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-white text-xl">加载中...</div>
      </div>
    )
  }

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

        {/* 对局表格 */}
        <div className="bg-gray-800/50 rounded-lg border border-gray-700 overflow-hidden">
          <GameResultTable games={games} getHeroName={getHeroName} />
        </div>
      </div>
    </div>
  )
}

export default HistoryPage