import React from 'react'
import HeroCard from './HeroCard'
import type { Hero } from '../types'

interface BanPickBoardProps {
  // 英雄列表
  heroes: Hero[]
  heroesByLane: Record<string, Hero[]>

  // BP 状态
  blueBanned: number[]
  redBanned: number[]
  bluePicked: number[]
  redPicked: number[]
  currentTurn: number
  action: string
  side: string

  // 操作回调
  onBanHero?: (hero: Hero) => void
  onPickHero?: (hero: Hero) => void

  // 当前用户所在方
  mySide: string
}

const LANE_ORDER = ['对抗路', '中路', '发育路', '打野', '游走']

const BanPickBoard: React.FC<BanPickBoardProps> = ({
  heroes,
  heroesByLane,
  blueBanned,
  redBanned,
  bluePicked,
  redPicked,
  currentTurn,
  action,
  side,
  onBanHero,
  onPickHero,
  mySide,
}) => {
  // 获取英雄的 owner 状态
  const getHeroOwner = (heroId: number): string | null => {
    if (blueBanned.includes(heroId)) return 'banned_blue'
    if (redBanned.includes(heroId)) return 'banned_red'
    if (bluePicked.includes(heroId)) return 'picked_blue'
    if (redPicked.includes(heroId)) return 'picked_red'
    return null
  }

  // 判断英雄是否可操作
  const isHeroActionable = (heroId: number): boolean => {
    return getHeroOwner(heroId) === null
  }

  // 处理英雄点击
  const handleHeroClick = (hero: Hero) => {
    if (!isHeroActionable(hero.id)) return
    // 只有当前操作方才能操作
    if (side !== mySide) return

    if (action === 'ban' && onBanHero) {
      onBanHero(hero)
    } else if (action === 'pick' && onPickHero) {
      onPickHero(hero)
    }
  }

  // 查找英雄名称
  const getHeroName = (heroId: number): string => {
    return heroes.find((h) => h.id === heroId)?.name || `英雄${heroId}`
  }

  return (
    <div className="w-full">
      {/* 状态提示 */}
      <div className="text-center mb-4">
        <div className="text-lg font-bold text-yellow-400">
          第 {currentTurn}/20 步 —{' '}
          {side === 'blue' ? '🔵 蓝方' : '🔴 红方'}{' '}
          {action === 'ban' ? '禁用' : '选择'}
        </div>
        {mySide && (
          <div className="text-sm text-gray-400 mt-1">
            {side === mySide
              ? '⏳ 轮到你了'
              : '👀 等待对方操作'}
          </div>
        )}
      </div>

      {/* 已 Ban 区域 */}
      <div className="mb-4">
        <div className="text-sm font-bold text-gray-300 mb-2">🚫 已禁用</div>
        <div className="flex gap-2 flex-wrap">
          {/* 蓝方禁用 */}
          {blueBanned.length > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-blue-400">蓝:</span>
              {blueBanned.map((id) => (
                <div
                  key={`bb-${id}`}
                  className="w-12 h-12 rounded border-2 border-blue-500/50 bg-blue-900/30 flex items-center justify-center"
                >
                  <span className="text-xs text-blue-300 truncate">
                    {getHeroName(id)}
                  </span>
                </div>
              ))}
            </div>
          )}
          {/* 红方禁用 */}
          {redBanned.length > 0 && (
            <div className="flex items-center gap-2 ml-4">
              <span className="text-xs text-red-400">红:</span>
              {redBanned.map((id) => (
                <div
                  key={`rb-${id}`}
                  className="w-12 h-12 rounded border-2 border-red-500/50 bg-red-900/30 flex items-center justify-center"
                >
                  <span className="text-xs text-red-300 truncate">
                    {getHeroName(id)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* 已 Pick 区域 */}
      <div className="mb-4">
        <div className="text-sm font-bold text-gray-300 mb-2">✅ 已选择</div>
        <div className="flex gap-4">
          {/* 蓝方阵容 */}
          <div className="flex-1 bg-blue-900/20 rounded-lg p-3 border border-blue-500/30">
            <div className="text-xs text-blue-400 mb-2">🔵 蓝方阵容</div>
            <div className="flex gap-2 flex-wrap">
              {bluePicked.map((id) => (
                <div
                  key={`bp-${id}`}
                  className="w-12 h-12 rounded border-2 border-blue-400/60 bg-blue-800/40 flex items-center justify-center"
                >
                  <span className="text-xs text-blue-200 truncate">
                    {getHeroName(id)}
                  </span>
                </div>
              ))}
              {bluePicked.length === 0 && (
                <span className="text-xs text-gray-500">等待选择...</span>
              )}
            </div>
          </div>
          {/* 红方阵容 */}
          <div className="flex-1 bg-red-900/20 rounded-lg p-3 border border-red-500/30">
            <div className="text-xs text-red-400 mb-2">🔴 红方阵容</div>
            <div className="flex gap-2 flex-wrap">
              {redPicked.map((id) => (
                <div
                  key={`rp-${id}`}
                  className="w-12 h-12 rounded border-2 border-red-400/60 bg-red-800/40 flex items-center justify-center"
                >
                  <span className="text-xs text-red-200 truncate">
                    {getHeroName(id)}
                  </span>
                </div>
              ))}
              {redPicked.length === 0 && (
                <span className="text-xs text-gray-500">等待选择...</span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* 英雄选择区（按分路） */}
      <div className="grid grid-cols-5 gap-3">
        {LANE_ORDER.map((lane) => (
          <div key={lane} className="bg-gray-800/50 rounded-lg p-2 border border-gray-700">
            <div className="text-xs font-bold text-gray-400 mb-2 text-center">
              {lane}
            </div>
            <div className="flex flex-wrap gap-1 justify-center">
              {(heroesByLane[lane] || []).map((hero) => (
                <HeroCard
                  key={hero.id}
                  hero={hero}
                  disabled={!isHeroActionable(hero.id)}
                  owner={getHeroOwner(hero.id)}
                  onClick={handleHeroClick}
                  size="small"
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default BanPickBoard