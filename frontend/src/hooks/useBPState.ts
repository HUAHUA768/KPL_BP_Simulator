import { useMemo } from 'react'
import { useBPStore } from '../store/bpStore'
import type { Hero } from '../types'

// useBPState — 从 store 中派生 BP 状态，供组件使用
export function useBPState() {
  const store = useBPStore()

  // 获取英雄名称（从 heroList 和 hero state 中查找）
  const getHeroName = (heroId: number): string => {
    const hero = store.heroList.find((h) => h.id === heroId)
    return hero?.name || `英雄${heroId}`
  }

  const getHeroIcon = (heroId: number): string => {
    const hero = store.heroList.find((h) => h.id === heroId)
    return hero?.iconPath || ''
  }

  // 按分路过滤英雄
  const heroesByLane = useMemo(() => {
    const lanes: Record<string, Hero[]> = {
      '对抗路': [],
      '中路': [],
      '发育路': [],
      '打野': [],
      '游走': [],
    }
    for (const hero of store.heroList) {
      for (const lane of hero.lanes) {
        if (lanes[lane]) {
          lanes[lane].push(hero)
        }
      }
    }
    return lanes
  }, [store.heroList])

  // 判断英雄是否可用
  const isHeroAvailable = (heroId: number): boolean => {
    // 如果已被 ban 或 pick，不可用
    if (
      store.blueBanned.includes(heroId) ||
      store.redBanned.includes(heroId) ||
      store.bluePicked.includes(heroId) ||
      store.redPicked.includes(heroId)
    ) {
      return false
    }
    return true
  }

  // 获取英雄被谁禁用/选择
  const getHeroOwner = (heroId: number): string | null => {
    if (store.blueBanned.includes(heroId)) return 'banned_blue'
    if (store.redBanned.includes(heroId)) return 'banned_red'
    if (store.bluePicked.includes(heroId)) return 'picked_blue'
    if (store.redPicked.includes(heroId)) return 'picked_red'
    return null
  }

  // 当前应该操作的一方
  const isMyTurn = (mySide: string): boolean => {
    return store.side === mySide
  }

  return {
    ...store,
    getHeroName,
    getHeroIcon,
    heroesByLane,
    isHeroAvailable,
    getHeroOwner,
    isMyTurn,
  }
}