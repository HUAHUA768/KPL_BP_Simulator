import React, { useEffect, useState } from 'react'
import { useParams, useSearchParams, useNavigate } from 'react-router-dom'
import BanPickBoard from '../components/BanPickBoard'
import { useWebSocket } from '../hooks/useWebSocket'
import { useBPState } from '../hooks/useBPState'
import { getRoomStatus, getHeroes, banHero, pickHero } from '../services/api'
import { useBPStore } from '../store/bpStore'
import type { Hero } from '../types'

const BPRoom: React.FC = () => {
  const { id: roomId } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const mySide = searchParams.get('side') || 'blue'

  const bpState = useBPState()
  const setHeroList = useBPStore((s) => s.setHeroList)
  const updateBPState = useBPStore((s) => s.updateBPState)

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // WebSocket 连接
  const { ban: wsBan, pick: wsPick } = useWebSocket(roomId || '', mySide)

  // 初始化：加载英雄列表 + 获取房间状态
  useEffect(() => {
    if (!roomId) return

    Promise.all([getHeroes(), getRoomStatus(roomId)])
      .then(([heroes, state]) => {
        setHeroList(heroes)
        updateBPState(state)
        setLoading(false)
      })
      .catch((err) => {
        setError('加载失败: ' + (err?.response?.data?.message || err.message))
        setLoading(false)
      })
  }, [roomId, setHeroList, updateBPState])

  // 处理 ban 操作
  const handleBanHero = async (hero: Hero) => {
    if (!roomId) return
    try {
      // 通过 HTTP API 执行（也通过 WebSocket 广播）
      const state = await banHero(roomId, hero.id)
      updateBPState(state)
      // 同时通过 WS 发送
      wsBan(hero.id)
    } catch (err: any) {
      setError(err?.response?.data?.message || '操作失败')
    }
  }

  // 处理 pick 操作
  const handlePickHero = async (hero: Hero) => {
    if (!roomId) return
    try {
      const state = await pickHero(roomId, hero.id)
      updateBPState(state)
      wsPick(hero.id)
    } catch (err: any) {
      setError(err?.response?.data?.message || '操作失败')
    }
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
      {/* 顶部导航 */}
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center justify-between mb-4">
          <button
            className="text-gray-400 hover:text-white transition-colors"
            onClick={() => navigate('/')}
          >
            ← 返回首页
          </button>
          <div className="text-gray-400 text-sm">
            房间: <span className="text-yellow-400">{roomId}</span>
            &nbsp;|&nbsp; 你:
            <span className={mySide === 'blue' ? 'text-blue-400' : 'text-red-400'}>
              {mySide === 'blue' ? ' 🔵 蓝方' : ' 🔴 红方'}
            </span>
          </div>
        </div>

        {/* 错误提示 */}
        {error && (
          <div className="bg-red-900/40 border border-red-500 text-red-300 rounded px-4 py-2 mb-4 text-center">
            {error}
            <button
              className="ml-2 text-red-400 hover:text-red-200 underline"
              onClick={() => setError(null)}
            >
              关闭
            </button>
          </div>
        )}

        {/* BP 完成提示 */}
        {bpState.status === 'finished' && (
          <div className="bg-green-900/40 border border-green-500 text-green-300 rounded px-4 py-2 mb-4 text-center text-lg font-bold">
            🎉 BP 完成！双方阵容已确定
          </div>
        )}

        {/* BP 棋盘 */}
        <BanPickBoard
          heroes={bpState.heroList}
          heroesByLane={bpState.heroesByLane}
          blueBanned={bpState.blueBanned}
          redBanned={bpState.redBanned}
          bluePicked={bpState.bluePicked}
          redPicked={bpState.redPicked}
          currentTurn={bpState.currentTurn}
          action={bpState.action}
          side={bpState.side}
          onBanHero={handleBanHero}
          onPickHero={handlePickHero}
          mySide={mySide}
        />
      </div>
    </div>
  )
}

export default BPRoom