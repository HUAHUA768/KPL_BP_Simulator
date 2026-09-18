import React, { useMemo } from 'react'
import HeroCard, { HeroIcon } from './HeroCard'
import type { Hero } from '../types'

interface BanPickBoardProps {
  // 英雄池（当前分路过滤后用于展示）
  heroes: Hero[]
  // 完整英雄列表（用于查名字 / 图标，避免分路过滤后查不到）
  allHeroes?: Hero[]

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

type Owner = 'banned_blue' | 'banned_red' | 'picked_blue' | 'picked_red'

// 每行英雄数量
const HEROES_PER_ROW = 10
// 每方 Pick 槽位数量（KPL 每方固定 5 个）
const PICKS_PER_SIDE = 5

const SLOT_TONE: Record<Owner, { box: string; text: string }> = {
  banned_blue: { box: 'border-blue-500/60 bg-blue-900/30', text: 'text-blue-300' },
  banned_red: { box: 'border-red-500/60 bg-red-900/30', text: 'text-red-300' },
  picked_blue: { box: 'border-blue-400/70 bg-blue-800/40', text: 'text-blue-200' },
  picked_red: { box: 'border-red-400/70 bg-red-800/40', text: 'text-red-200' },
}

// ---------------------------------------------------------------------------
// HeroSlot — 已禁用 / 已选择区统一的固定尺寸槽位
// 所有槽位结构完全一致（正方形图标 + 固定名称行），因此图标大小绝对统一。
// ---------------------------------------------------------------------------
const HeroSlot: React.FC<{ name: string; icon?: string; owner: Owner }> = ({
  name,
  icon,
  owner,
}) => {
  const tone = SLOT_TONE[owner]
  return (
    <div
      title={name}
      className={`flex w-14 shrink-0 flex-col items-center overflow-hidden rounded-md border-2 ${tone.box}`}
    >
      <div className="relative w-full overflow-hidden rounded-sm bg-gray-900/60 aspect-square">
        <HeroIcon src={icon} name={name} className="absolute inset-0 h-full w-full" />
      </div>
      <span
        className={`w-full truncate px-0.5 text-center text-[9px] leading-tight ${tone.text}`}
      >
        {name}
      </span>
    </div>
  )
}

// 空槽位：与 HeroSlot 结构一致，保证占位时尺寸不跳动
const EmptySlot: React.FC = () => (
  <div className="flex w-14 shrink-0 flex-col items-center overflow-hidden rounded-md border-2 border-dashed border-gray-700/70 bg-gray-900/20">
    <div className="w-full aspect-square" />
    <span className="w-full px-0.5 text-center text-[9px] leading-tight text-transparent">
      ·
    </span>
  </div>
)

const BanPickBoard: React.FC<BanPickBoardProps> = ({
  heroes,
  allHeroes,
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
  // 查名字 / 图标始终使用完整列表
  const heroMap = useMemo(() => {
    const source = allHeroes && allHeroes.length > 0 ? allHeroes : heroes
    const map = new Map<number, Hero>()
    source.forEach((h) => map.set(h.id, h))
    return map
  }, [allHeroes, heroes])

  const getHeroName = (heroId: number): string =>
    heroMap.get(heroId)?.name || `英雄${heroId}`

  const getHeroIcon = (heroId: number): string | undefined =>
    heroMap.get(heroId)?.iconPath

  // 获取英雄的 owner 状态
  const getHeroOwner = (heroId: number): Owner | null => {
    if (blueBanned.includes(heroId)) return 'banned_blue'
    if (redBanned.includes(heroId)) return 'banned_red'
    if (bluePicked.includes(heroId)) return 'picked_blue'
    if (redPicked.includes(heroId)) return 'picked_red'
    return null
  }

  const isHeroActionable = (heroId: number): boolean => getHeroOwner(heroId) === null

  // 是否轮到本方操作
  const isMyTurnNow = side === mySide && (action === 'ban' || action === 'pick')

  const handleHeroClick = (hero: Hero) => {
    if (!isHeroActionable(hero.id)) return
    if (!isMyTurnNow) return

    if (action === 'ban' && onBanHero) {
      onBanHero(hero)
    } else if (action === 'pick' && onPickHero) {
      onPickHero(hero)
    }
  }

  // Pick 槽位补足到每方 5 个
  const bluePickSlots = useMemo<(number | null)[]>(
    () =>
      Array.from({ length: PICKS_PER_SIDE }, (_, i) => bluePicked[i] ?? null),
    [bluePicked],
  )
  const redPickSlots = useMemo<(number | null)[]>(
    () => Array.from({ length: PICKS_PER_SIDE }, (_, i) => redPicked[i] ?? null),
    [redPicked],
  )

  // Ban 槽位：两侧取较长的，保证两行对齐
  const banSlotCount = Math.max(blueBanned.length, redBanned.length)
  const blueBanSlots = Array.from({ length: banSlotCount }, (_, i) => blueBanned[i] ?? null)
  const redBanSlots = Array.from({ length: banSlotCount }, (_, i) => redBanned[i] ?? null)

  const renderBanRow = (
    ids: (number | null)[],
    owner: Owner,
    label: string,
    labelClass: string,
  ) => (
    <div className="flex items-center justify-center gap-2">
      <span className={`w-8 shrink-0 text-right text-xs font-bold ${labelClass}`}>{label}</span>
      <div className="flex flex-wrap items-center justify-center gap-1.5">
        {ids.map((id, idx) =>
          id === null ? (
            <EmptySlot key={`empty-${owner}-${idx}`} />
          ) : (
            <HeroSlot
              key={`${owner}-${id}`}
              name={getHeroName(id)}
              icon={getHeroIcon(id)}
              owner={owner}
            />
          ),
        )}
      </div>
    </div>
  )

  return (
    <div className="mx-auto w-full max-w-7xl">
      {/* 状态提示 */}
      <div className="mb-4 text-center">
        <div className="text-lg font-bold text-yellow-400">
          第 {currentTurn}/20 步 — {side === 'blue' ? '🔵 蓝方' : '🔴 红方'}{' '}
          {action === 'ban' ? '禁用' : '选择'}
        </div>
        {mySide && (
          <div
            className={`mt-1 text-sm ${isMyTurnNow ? 'font-bold text-green-400' : 'text-gray-400'}`}
          >
            {isMyTurnNow ? ' 轮到你了，请点击下方英雄' : '👀 等待对方操作'}
          </div>
        )}
      </div>

      {/* 已 Ban 区域 */}
      <div className="mb-5">
        <div className="mb-2 text-center text-sm font-bold text-gray-300">🚫 已禁用</div>
        {banSlotCount === 0 ? (
          <div className="py-2 text-center text-xs text-gray-500">暂无禁用</div>
        ) : (
          <div className="flex flex-col items-center gap-1.5">
            {renderBanRow(blueBanSlots, 'banned_blue', '蓝', 'text-blue-400')}
            {renderBanRow(redBanSlots, 'banned_red', '红', 'text-red-400')}
          </div>
        )}
      </div>

      {/* 已 Pick 区域 */}
      <div className="mb-5">
        <div className="mb-2 text-center text-sm font-bold text-gray-300">✅ 已选择</div>
        <div className="flex flex-col justify-center gap-3 md:flex-row">
          {/* 蓝方阵容 */}
          <div className="flex-1 rounded-lg border border-blue-500/30 bg-blue-900/20 p-3">
            <div className="mb-2 text-center text-xs font-bold text-blue-400">🔵 蓝方阵容</div>
            <div className="flex flex-wrap items-center justify-center gap-1.5">
              {bluePickSlots.map((id, idx) =>
                id === null ? (
                  <EmptySlot key={`empty-blue-pick-${idx}`} />
                ) : (
                  <HeroSlot
                    key={`picked_blue-${id}`}
                    name={getHeroName(id)}
                    icon={getHeroIcon(id)}
                    owner="picked_blue"
                  />
                ),
              )}
            </div>
          </div>
          {/* 红方阵容 */}
          <div className="flex-1 rounded-lg border border-red-500/30 bg-red-900/20 p-3">
            <div className="mb-2 text-center text-xs font-bold text-red-400">🔴 红方阵容</div>
            <div className="flex flex-wrap items-center justify-center gap-1.5">
              {redPickSlots.map((id, idx) =>
                id === null ? (
                  <EmptySlot key={`empty-red-pick-${idx}`} />
                ) : (
                  <HeroSlot
                    key={`picked_red-${id}`}
                    name={getHeroName(id)}
                    icon={getHeroIcon(id)}
                    owner="picked_red"
                  />
                ),
              )}
            </div>
          </div>
        </div>
      </div>

      {/* 英雄选择区（固定每行 10 个） */}
      <div>
        <div className="mb-2 text-center text-sm font-bold text-gray-300">
          🗡️ 英雄池（每行 {HEROES_PER_ROW} 个）
        </div>
        <div className="rounded-lg border border-gray-700 bg-gray-800/50 p-3">
          {heroes.length === 0 ? (
            <div className="py-8 text-center text-sm text-gray-500">该分路暂无英雄</div>
          ) : (
            <div className="overflow-x-auto">
              <ul className="grid min-w-[760px] grid-cols-10 gap-1.5">
                {heroes.map((hero) => (
                  <li key={hero.id} className="min-w-0">
                    <HeroCard
                      hero={hero}
                      disabled={!isHeroActionable(hero.id)}
                      blocked={!isMyTurnNow}
                      owner={getHeroOwner(hero.id)}
                      onClick={handleHeroClick}
                    />
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default BanPickBoard