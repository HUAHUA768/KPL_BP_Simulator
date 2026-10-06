// ============================================================================
// actor — 有效操作方判定（双人扩展的唯一接缝）
// 全项目唯一允许出现"操作方判定"逻辑的地方。
// ============================================================================

/** 交互模式：solo = 一人分饰两角（默认）；dual = 双人各持一方（预留） */
export type Mode = 'solo' | 'dual'

/**
 * 谁是有效操作方：
 * - solo：引擎当前回合方（stateSide），门控恒通过——谁回合谁就是"我"
 * - dual：URL 声明方（urlSide），保留旧的"非本方回合只读等待"语义
 */
export function resolveActor(stateSide: string, mode: Mode, urlSide: string): string {
  return mode === 'dual' ? urlSide : stateSide
}

/** 解析 ?mode= 查询参数，非法值一律回退 solo */
export function parseMode(search: string): Mode {
  const params = new URLSearchParams(search)
  return params.get('mode') === 'dual' ? 'dual' : 'solo'
}
