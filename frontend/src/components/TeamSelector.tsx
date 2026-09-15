import React, { useState, useEffect } from 'react'
import { getTeams } from '../services/api'
import type { Team } from '../types'

interface TeamSelectorProps {
  onSelect: (blueTeam: Team, redTeam: Team) => void
  loading?: boolean
}

const TeamSelector: React.FC<TeamSelectorProps> = ({ onSelect, loading = false }) => {
  const [teams, setTeams] = useState<Team[]>([])
  const [blueTeam, setBlueTeam] = useState<Team | null>(null)
  const [redTeam, setRedTeam] = useState<Team | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getTeams()
      .then(setTeams)
      .catch(() => setError('加载战队列表失败'))
  }, [])

  const handleStart = () => {
    if (!blueTeam || !redTeam) {
      setError('请选择两支战队')
      return
    }
    if (blueTeam.id === redTeam.id) {
      setError('不能选择相同的战队')
      return
    }
    setError(null)
    onSelect(blueTeam, redTeam)
  }

  return (
    <div className="bg-gray-800/70 rounded-lg p-6 border border-gray-700 max-w-md mx-auto">
      <h2 className="text-xl font-bold text-white mb-4 text-center">
        选择对战队伍
      </h2>

      {error && (
        <div className="bg-red-900/40 border border-red-500 text-red-300 rounded px-3 py-2 mb-4 text-sm">
          {error}
        </div>
      )}

      {/* 蓝方选择 */}
      <div className="mb-4">
        <label className="block text-sm text-blue-400 mb-2">🔵 蓝方 (先Ban方)</label>
        <select
          className="w-full bg-gray-900 border border-gray-600 rounded-lg px-3 py-2 text-white focus:border-blue-500 focus:outline-none"
          value={blueTeam?.id || ''}
          onChange={(e) => {
            const id = Number(e.target.value)
            const team = teams.find((t) => t.id === id)
            setBlueTeam(team || null)
          }}
        >
          <option value="">-- 选择蓝方 --</option>
          {teams.map((team) => (
            <option key={team.id} value={team.id}>
              {team.name}
            </option>
          ))}
        </select>
      </div>

      {/* VS */}
      <div className="text-center text-2xl font-bold text-yellow-400 mb-4">
        VS
      </div>

      {/* 红方选择 */}
      <div className="mb-6">
        <label className="block text-sm text-red-400 mb-2">🔴 红方 (后Ban方)</label>
        <select
          className="w-full bg-gray-900 border border-gray-600 rounded-lg px-3 py-2 text-white focus:border-red-500 focus:outline-none"
          value={redTeam?.id || ''}
          onChange={(e) => {
            const id = Number(e.target.value)
            const team = teams.find((t) => t.id === id)
            setRedTeam(team || null)
          }}
        >
          <option value="">-- 选择红方 --</option>
          {teams.map((team) => (
            <option key={team.id} value={team.id}>
              {team.name}
            </option>
          ))}
        </select>
      </div>

      {/* 开始按钮 */}
      <button
        className="w-full bg-yellow-500 hover:bg-yellow-400 text-black font-bold py-3 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        disabled={!blueTeam || !redTeam || loading}
        onClick={handleStart}
      >
        {loading ? '创建中...' : '开始 BP'}
      </button>
    </div>
  )
}

export default TeamSelector