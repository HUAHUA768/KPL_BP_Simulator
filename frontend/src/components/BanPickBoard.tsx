import React, { useMemo } from 'react'
import HeroCard, { HeroIcon } from './HeroCard'
import type { Hero, TeamBrief } from '../types'

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

  // 有效操作方（由调用方经 resolveActor 算好；组件不再自行判断身份）
  actorSide: string

  // 双方队名 + 队标（问题1）；缺失时降级显示"🔵 蓝方 / 红方 🔴"
  blueTeam?: TeamBrief | null
  redTeam?: TeamBrief | null
}

type SideName = 'blue' | 'red'
type ActionName = 'ban' | 'pick'

// ---------------------------------------------------------------------------
// 20 步 [side, action] 序列表
// 与 backend/internal/service/bp_engine.go 的 getTurnInfo 逐行核对抄录，
// 后端状态机是唯一事实来源，本表只是其前端镜像。
// ---------------------------------------------------------------------------
const TURN_SEQUENCE: ReadonlyArray<readonly [SideName, ActionName]> = [
  ['blue', 'ban'], // 1  蓝ban1
  ['red', 'ban'], // 2  红ban1
  ['blue', 'ban'], // 3  蓝ban2
  ['red', 'ban'], // 4  红ban2
  ['blue', 'pick'], // 5  蓝pick1
  ['red', 'pick'], // 6  红pick1
  ['red', 'pick'], // 7  红pick2
  ['blue', 'pick'], // 8  蓝pick2
  ['blue', 'pick'], // 9  蓝pick3
  ['red', 'pick'], // 10 红pick3
  ['red', 'ban'], // 11 红ban3
  ['blue', 'ban'], // 12 蓝ban4
  ['red', 'ban'], // 13 红ban4
  ['blue', 'ban'], // 14 蓝ban5
  ['red', 'ban'], // 15 红ban5
  ['blue', 'ban'], // 16 蓝ban6
  ['red', 'pick'], // 17 红pick4
  ['blue', 'pick'], // 18 蓝pick4
  ['blue', 'pick'], // 19 蓝pick5
  ['red', 'pick'], // 20 红pick5
]

// 从 currentTurn 起，向后连续的"同 side 同 action"步数 k
// —— 对应当前方应同时闪烁的空槽位个数（如 18→19 蓝pick4/蓝pick5，k=2）
export function countConsecutiveSameSlots(side: string, action: string, currentTurn: number): number {
  let k = 0
  for (let t = currentTurn; t <= TURN_SEQUENCE.length; t++) {
    const [s, a] = TURN_SEQUENCE[t - 1]
    if (s !== side || a !== action) break
    k++
  }
  return k
}

// 当前步是该方此动作的第几手（如蓝ban2 → 2）；无效步返回 0
export function turnActionIndex(currentTurn: number): number {
  if (currentTurn < 1 || currentTurn > TURN_SEQUENCE.length) return 0
  const [s, a] = TURN_SEQUENCE[currentTurn - 1]
  let idx = 0
  for (let t = 1; t <= currentTurn; t++) {
    const [ts, ta] = TURN_SEQUENCE[t - 1]
    if (ts === s && ta === a) idx++
  }
  return idx
}

type Owner = 'banned_blue' | 'banned_red' | 'picked_blue' | 'picked_red'

// 每行英雄数量
const HEROES_PER_ROW = 10
// 每方 Ban / Pick 槽位数量（由 20 步序列表决定：双方各 ban 5、各 pick 5，固定不变）
const BANS_PER_SIDE = 5
const PICKS_PER_SIDE = 5

// ---------------------------------------------------------------------------
// 槽位尺寸分级（问题2）：ban 位相对小、pick 位相对大
// 禁用只是"标记"，选用才是"成果"，用体量差建立视觉层级。
//
// ⚠️ HeroSlot 与 EmptySlot 必须使用同一张表 —— 两者尺寸若不一致，
//    槽位从"空"到"被填充"的瞬间会发生尺寸跳动，破坏零跳动布局。
// ---------------------------------------------------------------------------
type SlotSize = 'ban' | 'pick'

const SLOT_SIZE: Record<SlotSize, { box: string; name: string }> = {
  ban: { box: 'w-11', name: 'text-[8px]' }, // 2.75rem —— 相对小
  pick: { box: 'w-16', name: 'text-[9px]' }, // 4rem    —— 相对大
}

const SLOT_TONE: Record<Owner, { box: string; text: string }> = {
  banned_blue: { box: 'border-blue-500/60 bg-blue-900/30', text: 'text-blue-300' },
  banned_red: { box: 'border-red-500/60 bg-red-900/30', text: 'text-red-300' },
  picked_blue: { box: 'border-blue-400/70 bg-blue-800/40', text: 'text-blue-200' },
  picked_red: { box: 'border-red-400/70 bg-red-800/40', text: 'text-red-200' },
}

// ---------------------------------------------------------------------------
// HeroSlot — 已禁用 / 已选择区统一的固定尺寸槽位
// 同一 size 下的槽位结构完全一致（正方形图标 + 固定名称行），因此图标大小绝对统一。
// ---------------------------------------------------------------------------
const HeroSlot: React.FC<{ name: string; icon?: string; owner: Owner; size: SlotSize }> = ({
  name,
  icon,
  owner,
  size,
}) => {
  const tone = SLOT_TONE[owner]
  const dim = SLOT_SIZE[size]
  return (
    <div
      title={name}
      className={`flex ${dim.box} shrink-0 flex-col items-center overflow-hidden rounded-md border-2 ${tone.box}`}
    >
      <div className="relative w-full overflow-hidden rounded-sm bg-gray-900/60 aspect-square">
        <HeroIcon src={icon} name={name} className="absolute inset-0 h-full w-full" />
      </div>
      <span
        className={`w-full truncate px-0.5 text-center leading-tight ${dim.name} ${tone.text}`}
      >
        {name}
      </span>
    </div>
  )
}

// 回合方主题色：横幅 / 棋盘外框 glow / 槽位闪烁颜色均随当前回合方整页切换
const SIDE_TONE: Record<
  SideName,
  { banner: string; bannerText: string; glow: string; blinkColor: string }
> = {
  blue: {
    banner:
      'border-blue-500/60 bg-gradient-to-r from-blue-900/70 via-blue-800/40 to-blue-900/70',
    bannerText: 'text-blue-200',
    glow: 'border-blue-500/50 shadow-[0_0_40px_rgba(59,130,246,0.25)]',
    blinkColor: '#3b82f6',
  },
  red: {
    banner:
      'border-red-500/60 bg-gradient-to-r from-red-900/70 via-red-800/40 to-red-900/70',
    bannerText: 'text-red-200',
    glow: 'border-red-500/50 shadow-[0_0_40px_rgba(239,68,68,0.25)]',
    blinkColor: '#ef4444',
  },
}

// 空槽位：与 HeroSlot 结构一致，保证占位时尺寸不跳动。
// blinking 时挂 slot-blinking class（keyframes 见 index.css），
// 颜色经 CSS 变量 --blink-color 按回合方区分；动作完成后槽位被 HeroSlot
// 取代，动画自然消失，无需任何"停止动画"逻辑。
const EmptySlot: React.FC<{ blinking?: boolean; tone?: SideName; size: SlotSize }> = ({
  blinking = false,
  tone,
  size,
}) => {
  const dim = SLOT_SIZE[size]
  return (
    <div
      className={`flex ${dim.box} shrink-0 flex-col items-center overflow-hidden rounded-md border-2 border-dashed bg-gray-900/20 ${
        blinking ? 'slot-blinking' : 'border-gray-700/70'
      }`}
      style={
        blinking && tone
          ? ({ '--blink-color': SIDE_TONE[tone].blinkColor } as React.CSSProperties)
          : undefined
      }
    >
      <div className="w-full aspect-square" />
      <span
        className={`w-full px-0.5 text-center leading-tight text-transparent ${dim.name}`}
      >
        ·
      </span>
    </div>
  )
}

// ---------------------------------------------------------------------------
// TeamHeader — 阵营标题：队标 + 队名（问题1）
// 无队伍信息时降级为"🔵 蓝方 / 红方 🔴"，保证任何情况下都能看出红蓝归属。
// 队标加载失败时保留占位块（而非 display:none），避免问题被静默掩盖。
// ---------------------------------------------------------------------------
const TeamHeader: React.FC<{
  team?: TeamBrief | null
  fallback: string
  mirrored: boolean
  labelClass: string
}> = ({ team, fallback, mirrored, labelClass }) => {
  const [logoFailed, setLogoFailed] = React.useState(false)
  const showLogo = !!team?.logoUrl && !logoFailed

  return (
    <div
      className={`mb-2 flex items-center gap-2 ${mirrored ? 'flex-row-reverse' : ''}`}
      title={team?.name || fallback}
    >
      {showLogo ? (
        <img
          src={team!.logoUrl}
          alt={team!.name}
          onError={() => setLogoFailed(true)}
          className="h-7 w-7 shrink-0 rounded object-contain"
        />
      ) : (
        /* 无队标（或加载失败）：灰色占位块，保持标题行高度稳定 */
        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded bg-gray-700/60 text-[10px] text-gray-400">
          {team?.name?.slice(0, 1) || '队'}
        </span>
      )}
      <span className={`truncate text-sm font-bold ${labelClass}`}>
        {team?.name || fallback}
      </span>
    </div>
  )
}

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
  actorSide,
  blueTeam,
  redTeam,
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

  // 门控：solo 下 actorSide === side 恒真；dual 下即旧的等待语义
  const inProgress = action === 'ban' || action === 'pick'
  const isMyTurnNow = actorSide === side && inProgress
  const tone = side === 'red' ? SIDE_TONE.red : SIDE_TONE.blue

  const handleHeroClick = (hero: Hero) => {
    if (!isHeroActionable(hero.id)) return
    if (!isMyTurnNow) return

    if (action === 'ban' && onBanHero) {
      onBanHero(hero)
    } else if (action === 'pick' && onPickHero) {
      onPickHero(hero)
    }
  }

  // Ban / Pick 槽位：每方固定 5 + 5，布局从头到尾零跳动
  const fill = (ids: number[], n: number): (number | null)[] =>
    Array.from({ length: n }, (_, i) => ids[i] ?? null)

  const blueBanSlots = fill(blueBanned, BANS_PER_SIDE)
  const redBanSlots = fill(redBanned, BANS_PER_SIDE)
  const bluePickSlots = fill(bluePicked, PICKS_PER_SIDE)
  const redPickSlots = fill(redPicked, PICKS_PER_SIDE)

  // 槽位闪烁：当前方"接下来将被填充"的空槽位。
  // 向后看 turn 序列，连续同 side 同 action 共 k 步 → 接下来 k 个空槽位同时闪。
  const EMPTY_SET = new Set<number>()
  let blinkingIdx = EMPTY_SET
  if (inProgress) {
    const k = countConsecutiveSameSlots(side, action, currentTurn)
    const filled =
      action === 'ban'
        ? side === 'blue'
          ? blueBanned.length
          : redBanned.length
        : side === 'blue'
          ? bluePicked.length
          : redPicked.length
    blinkingIdx = new Set(Array.from({ length: k }, (_, i) => filled + i))
  }
  // 某一行槽位是否命中闪烁：仅当前回合方 + 当前动作类型的那一类槽位
  const blinkFor = (rowSide: SideName, slotAction: ActionName) =>
    side === rowSide && action === slotAction ? blinkingIdx : EMPTY_SET

  // 单一槽位行：label 可在左（蓝，靠右对齐）或右（红，镜像）
  // size 决定该行槽位体量（问题2：ban 小 / pick 大）
  const renderSlotRow = (
    ids: (number | null)[],
    owner: Owner,
    label: string,
    labelClass: string,
    blink: Set<number>,
    blinkTone: SideName,
    mirrored: boolean,
    size: SlotSize,
  ) => (
    <div className="flex items-center gap-2">
      {!mirrored && (
        <span className={`w-9 shrink-0 text-right text-xs font-bold ${labelClass}`}>
          {label}
        </span>
      )}
      <div className="flex flex-1 flex-wrap items-center justify-start gap-1.5">
        {ids.map((id, idx) =>
          id === null ? (
            <EmptySlot
              key={`empty-${owner}-${idx}`}
              blinking={blink.has(idx)}
              tone={blinkTone}
              size={size}
            />
          ) : (
            <HeroSlot
              key={`${owner}-${id}`}
              name={getHeroName(id)}
              icon={getHeroIcon(id)}
              owner={owner}
              size={size}
            />
          ),
        )}
      </div>
      {mirrored && (
        <span className={`w-9 shrink-0 text-left text-xs font-bold ${labelClass}`}>{label}</span>
      )}
    </div>
  )

  // 阵营卡片：上排 5 个 ban 槽 + 下排 5 个 pick 槽
  const renderSideCard = (cardSide: SideName) => {
    const mirrored = cardSide === 'red'
    const isBlue = cardSide === 'blue'
    const banSlots = isBlue ? blueBanSlots : redBanSlots
    const pickSlots = isBlue ? bluePickSlots : redPickSlots
    const banOwner: Owner = isBlue ? 'banned_blue' : 'banned_red'
    const pickOwner: Owner = isBlue ? 'picked_blue' : 'picked_red'
    const labelClass = isBlue ? 'text-blue-400' : 'text-red-400'

    return (
      <div
        className={`flex-1 rounded-lg border p-3 ${
          isBlue ? 'border-blue-500/30 bg-blue-900/20' : 'border-red-500/30 bg-red-900/20'
        }`}
      >
        {/* 阵营标题：队标 + 队名（问题1）；红方靠右镜像对称 */}
        <TeamHeader
          team={isBlue ? blueTeam : redTeam}
          fallback={isBlue ? '🔵 蓝方' : '红方 🔴'}
          mirrored={mirrored}
          labelClass={labelClass}
        />

        <div className="flex flex-col gap-2">
          {renderSlotRow(
            banSlots,
            banOwner,
            '禁用',
            `${labelClass} opacity-80`,
            blinkFor(cardSide, 'ban'),
            cardSide,
            mirrored,
            'ban',
          )}
          {renderSlotRow(
            pickSlots,
            pickOwner,
            '选择',
            labelClass,
            blinkFor(cardSide, 'pick'),
            cardSide,
            mirrored,
            'pick',
          )}
        </div>
      </div>
    )
  }

  return (
    <div
      className={`mx-auto w-full max-w-7xl rounded-xl border-2 p-3 transition-all duration-500 ${
        inProgress ? tone.glow : 'border-gray-800'
      }`}
    >
      {/* 状态横幅：主题色随当前回合方切换 */}
      {inProgress ? (
        <div
          className={`mb-4 rounded-lg border px-4 py-3 text-center transition-colors duration-500 ${tone.banner}`}
        >
          <div className={`text-lg font-bold ${tone.bannerText}`}>
            {isMyTurnNow ? '🎭 当前扮演：' : '👀 等待对方操作：'}
            {side === 'blue' ? '🔵 蓝方' : '🔴 红方'} · {action === 'ban' ? '禁用' : '选择'}第{' '}
            {turnActionIndex(currentTurn)} 手
          </div>
          <div className="mt-1 text-sm text-gray-300">
            第 {currentTurn}/20 步{isMyTurnNow ? ' — 轮到你了，请点击下方英雄' : ''}
          </div>
        </div>
      ) : (
        <div className="mb-4 rounded-lg border border-green-500/50 bg-green-900/30 px-4 py-3 text-center text-lg font-bold text-green-300">
          🎉 本局 BP 已完成
        </div>
      )}

      {/* 左右两列对称棋盘：左蓝（上 ban 下 pick）/ 右红镜像；窄屏纵向堆叠 */}
      <div className="mb-5 grid gap-3 md:grid-cols-2">
        {renderSideCard('blue')}
        {renderSideCard('red')}
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