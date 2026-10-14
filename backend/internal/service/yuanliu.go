package service

import (
	"strings"

	"kpl-bp-simulator/internal/model"
)

// ============================================================================
// 元流之子「五选一」锁定规则
// ============================================================================
//
// 规则来源：全局BP规则完善 — 元流之子专项。需求原文：
//
//	1. 五个分路的元流之子，A 队**本场比赛**只能选一个使用
//	   （一旦使用了元射，则其它四个元流都不能再选）
//	2. 基于第 1 点，若 A 队已使用元射，则 B 队不用再 ban 其它四个元流，
//	   但 B 队依然可以进行元流五选一，故而 A 队依然有 ban 五个元流的权力
//
// 本文件是该规则的**单一事实来源**：把「什么是元流」与「锁定怎么算」收敛为
// 一组纯函数，避免名字前缀判断散落到 game_service / room_handler / 前端三处。
//
// ⚠️ 三条经需求方确认的语义边界（任何与此冲突的实现都是 bug）：
//   - 锁定范围：**整场比赛累计**，不是单局
//   - ban 不计入：只有 pick 才触发锁定（ban 掉元射不会锁死其它四个）
//   - 锁是按队伍的：A 队被锁**不影响** B 队的五选一权力
//   - 只压 pick 不压 ban：已用过元流的队伍，依然对五个元流保有完整的 ban 权
// ============================================================================

// YuanliuPrefix 元流之子的名字前缀。
//
// ⚠️ 用「名字前缀」而非硬编码 ID：英雄 ID 由 auto_increment 决定，
// 不同环境（seed 重建 / 新增迁移）可能不同；名字才是稳定的业务事实。
// 当前库中五个元流 ID 为 33/60/83/105/132，但代码**不得**依赖这些数字。
const YuanliuPrefix = "元流之子"

// IsYuanliuHero 判断某英雄是否属于元流之子五选一组。
//
// 注意前缀带了「之子」，因此不会误伤「元歌」这类同以「元」开头但与元流无关的英雄。
func IsYuanliuHero(name string) bool {
	return strings.HasPrefix(strings.TrimSpace(name), YuanliuPrefix)
}

// YuanliuGroupIDs 从英雄列表里筛出元流组的英雄 ID。
//
// heroes 通常来自 heroRepo.ListActive()（按 id 升序），因此返回顺序稳定，便于测试断言。
// 返回值可能为空（英雄池中无元流英雄）—— 调用方必须能容忍这种情况。
// nil 元素与非法 ID（<=0）会被安全跳过，不 panic。
func YuanliuGroupIDs(heroes []*model.Hero) []int {
	ids := make([]int, 0, 5)
	for _, h := range heroes {
		if h == nil {
			continue
		}
		if h.ID > 0 && IsYuanliuHero(h.Name) {
			ids = append(ids, h.ID)
		}
	}
	return ids
}

// YuanliuLocked 计算「本队因已使用过元流而不可再选」的英雄集合。
//
// 规则公式（见方案 §2.2）：
//
//	T 本局能否 pick h ⇔ usedYuanliu(T) 为空 或 h ∈ usedYuanliu(T)
//
// 即：
//   - usedByTeam 为空          → 返回空集（该队仍有完整的五选一权力）
//   - usedByTeam 非空（已用过）→ 返回整组中**除已用过的那些之外**的全部 ID
//
// 之所以用「整组减去已用过的」而非「只锁其余四个」：需求 1 以元射举例，
// 但规则对五个元流一视同仁 —— 若某队历史 pick 过"坦克"，则其余四个（含元射）
// 同样应被锁。公式 `整组 - 已用过` 同时覆盖这两种表述。
//
// ⚠️ 注意：ban 不计入 usedByTeam（需求方确认）。调用方只应传入 pick 记录。
func YuanliuLocked(groupIDs []int, usedByTeam map[int]bool) map[int]bool {
	locked := make(map[int]bool)
	if len(usedByTeam) == 0 {
		return locked
	}
	for _, id := range groupIDs {
		if !usedByTeam[id] {
			locked[id] = true
		}
	}
	return locked
}
