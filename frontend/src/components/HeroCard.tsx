import React from 'react'
import type { Hero } from '../types'

interface HeroCardProps {
  hero: Hero
  disabled?: boolean
  owner?: string | null // 'banned_blue' | 'banned_red' | 'picked_blue' | 'picked_red' | null
  onClick?: (hero: Hero) => void
  size?: 'small' | 'medium' | 'large'
}

const HeroCard: React.FC<HeroCardProps> = ({
  hero,
  disabled = false,
  owner = null,
  onClick,
  size = 'medium',
}) => {
  const sizeClass = {
    small: 'w-14 h-14 text-xs',
    medium: 'w-16 h-16 text-sm',
    large: 'w-20 h-20 text-base',
  }[size]

  const getBorderColor = () => {
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

  const isClickable = !disabled && !owner && onClick

  return (
    <div
      className={`${sizeClass} rounded-lg border-2 ${getBorderColor()} 
        flex flex-col items-center justify-center cursor-pointer transition-all duration-200
        ${isClickable ? 'hover:scale-105' : ''}
        ${disabled ? 'opacity-40 cursor-not-allowed' : ''}
        ${owner ? 'opacity-60' : ''}`}
      onClick={() => isClickable && onClick?.(hero)}
      title={hero.name}
    >
      <img
        src={hero.iconPath}
        alt={hero.name}
        className="w-8 h-8 rounded object-cover"
        onError={(e) => {
          // 图片加载失败时显示占位符
          ;(e.target as HTMLImageElement).style.display = 'none'
        }}
      />
      <span className="text-white mt-0.5 truncate w-full text-center px-0.5">
        {hero.name}
      </span>
    </div>
  )
}

export default HeroCard