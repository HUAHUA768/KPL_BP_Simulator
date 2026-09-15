import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import TeamSelector from '../components/TeamSelector'
import { createRoom, getHeroes } from '../services/api'
import { useBPStore } from '../store/bpStore'
import type { Team } from '../types'

const HomePage: React.FC = () => {
  const navigate = useNavigate()
  const setRoomId = useBPStore((s) => s.setRoomId)
  const setHeroList = useBPStore((s) => s.setHeroList)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleTeamSelect = async (blueTeam: Team, redTeam: Team) => {
    setLoading(true)
    setError(null)

    try {
      // 1. 加载英雄列表（存入 store 供后续使用）
      const heroes = await getHeroes()
      setHeroList(heroes)

      // 2. 创建房间
      const room = await createRoom(blueTeam.id, redTeam.id)
      setRoomId(room.roomId)

      // 3. 跳转到 BP 房间
      navigate(`/room/${room.roomId}?side=blue`)
    } catch (err: any) {
      setError(err?.response?.data?.message || '创建房间失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-900 flex flex-col items-center justify-center p-4">
      {/* 标题 */}
      <div className="title-container flex flex-col items-center justify-center mb-8">
        <h1 className="text-4xl md:text-5xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-yellow-400 via-orange-500 to-red-500 mb-2">
          🏆 KPL BP 模拟器
        </h1>
        <p className="text-gray-400 text-center">
          王者荣耀职业联赛 Ban/Pick 流程模拟
        </p>
      </div>

      {/* 错误提示 */}
      {error && (
        <div className="bg-red-900/40 border border-red-500 text-red-300 rounded px-4 py-2 mb-4 max-w-md w-full text-center">
          {error}
        </div>
      )}

      {/* 队伍选择器 */}
      <TeamSelector onSelect={handleTeamSelect} loading={loading} />

      {/* 底部链接 */}
      <div className="mt-6">
        <button
          className="text-gray-500 hover:text-gray-300 text-sm underline transition-colors"
          onClick={() => navigate('/history')}
        >
          查看历史对局 →
        </button>
      </div>
    </div>
  )
}

export default HomePage