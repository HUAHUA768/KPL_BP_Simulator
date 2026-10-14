import React, { useState } from 'react'
import type { Hero } from '../types'

// ============================================================================
// HeroIcon — 统一的英雄图标
// 无论原图尺寸 / 宽高比如何，始终填满一个正方形并居中裁切，
// 加载失败时用占位块兜底（保持尺寸不变，避免布局塌陷 / 重叠）。
// ============================================================================
export const HeroIcon: React.FC<{
  src?: string
  name: string
  className?: string
}> = ({ src, name, className = '' }) => {
  const [failed, setFailed] = useState(false)

  if (!src || failed) {
    return (
      <div
        className={`flex items-center justify-center bg-gray-700/70 font-bold text-gray-300 ${className}`}
      >
        <span className="text-[10px] leading-none">{name.slice(0, 2)}</span>
      </div>
    )
  }

  return (
    <img
      src={src}
      alt={name}
      loading="lazy"
      draggable={false}
      className={`object-cover ${className}`}
      onError={() => setFailed(true)}
    />
  )
}

// ============================================================================
// HeroCard — 英雄池中的可选英雄卡片
// 卡片宽度由外层网格决定（每行 10 个），图标用 aspect-square 自适应，
// 因此所有英雄的图标尺寸完全一致，且不会超出单元格造成重叠。
// ============================================================================
interface HeroCardProps {
  hero: Hero
  disabled?: boolean
  /** 当前不是本方操作回合：仅弱化显示，不算“不可用” */
  blocked?: boolean
  owner?: string | null // 'banned_blue' | 'banned_red' | 'picked_blue' | 'picked_red' | null
  /**
   * 受全局BP规则限制（问题3/4）：当前操作方不可 ban/pick 此英雄。
   * 直接灰化 + 禁用光标，而非点击后才提示无效。
   */
  restricted?: boolean
  onClick?: (hero: Hero) => void
}

const HeroCard: React.FC<HeroCardProps> = ({
  hero,
  disabled = false,
  blocked = false,
  owner = null,
  restricted = false,
  onClick,
}) => {
  const getBorderColor = () => {
    // 受限英雄（问题3/4）：灰化边框，与"被 ban/pick 用过"的红蓝边框区分
    if (restricted && !owner) return 'border-gray-700 bg-gray-800/40'
    switch (owner) {
      case 'banned_blue':
        return 'border-blue-500 bg-blue-900/30'
      case 'banned_red':
        return 'border-red-500 bg-red-900/30'
      case 'picked_blue':
        return 'border-blue-400 bg-blue-800/40'
      case 'picked_red':
        return 'border-red-400 bg-red-800/40'
      default:
        return 'border-gray-600 bg-gray-800/60 hover:border-yellow-400 hover:bg-gray-700/80'
    }
  }

  const getBadge = () => {
    switch (owner) {
      case 'banned_blue':
      case 'banned_red':
        return { text: '禁', cls: 'bg-red-600/90 text-white' }
      case 'picked_blue':
        return { text: '蓝', cls: 'bg-blue-600/90 text-white' }
      case 'picked_red':
        return { text: '红', cls: 'bg-red-600/90 text-white' }
      default:
        return null
    }
  }

  const isClickable = !disabled && !blocked && !owner && !restricted && !!onClick
  const badge = getBadge()
  const isBanned = owner === 'banned_blue' || owner === 'banned_red'
  // 受限英雄需要灰化（问题3/4）：与"已被 ban/pick"同样灰，但加「限」角标区分
  const isDimmed = !!owner || restricted

  return (
    <button
      type="button"
      disabled={!isClickable}
      onClick={() => isClickable && onClick?.(hero)}
      title={
        restricted && !owner
          ? `${hero.name}（受全局BP规则限制，此方不可 ban/pick）`
          : hero.name
      }
      className={`
        flex w-full min-w-0 flex-col items-center overflow-hidden rounded-md border-2 p-0.5
        text-left transition-all duration-200 ${getBorderColor()}
        ${isClickable ? 'cursor-pointer hover:scale-105 hover:shadow-lg hover:shadow-yellow-500/20' : 'cursor-not-allowed'}
        ${disabled ? 'opacity-40' : ''}
        ${blocked ? 'opacity-60' : ''}
        ${restricted && !disabled && !blocked ? 'opacity-50' : ''}
      `}
    >
      {/* 正方形图标框：宽高一致，图片 object-cover 居中裁切 */}
      <div className="relative w-full overflow-hidden rounded bg-gray-900/60 aspect-square">
        {/* 已用 / 受限英雄整图灰化：彩色=可选，灰色=不可操作，形成强对比 */}
        <HeroIcon
          src={hero.iconPath}
          name={hero.name}
          className={`absolute inset-0 h-full w-full ${
            isDimmed ? 'grayscale brightness-50' : ''
          }`}
        />
        {/* 被 ban 的英雄叠加醒目红✕（保留「禁」角标，双重可辨） */}
        {isBanned && (
          <span className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <span className="text-4xl font-black leading-none text-red-500 drop-shadow-[0_0_4px_rgba(0,0,0,0.9)]">
              ✕
            </span>
          </span>
        )}
        {/* 受限英雄（问题3/4）：叠加「限」角标，说明是规则限制而非被用过 */}
        {restricted && !owner && (
          <span className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <span className="rounded bg-gray-900/80 px-1.5 py-0.5 text-sm font-black leading-none text-gray-300">
              限
            </span>
          </span>
        )}
        {badge && (
          <span
            className={`absolute right-0 top-0 rounded-bl px-1 text-[9px] font-bold leading-tight ${badge.cls}`}
          >
            {badge.text}
          </span>
        )}
      </div>
      {/* 固定高度的名称行：无论名称长短，卡片高度都一致 */}
      <span
        className={`mt-0.5 w-full truncate text-center text-[10px] leading-tight ${
          isDimmed ? 'text-gray-500' : 'text-white'
        }`}
      >
        {hero.name}
      </span>
    </button>
  )
}

export default HeroCard