package service

import (
	"testing"

	"kpl-bp-simulator/internal/model"
)

// ============================================================================
// 元流之子「五选一」锁定规则 — 纯函数单元测试
//
// 这些用例覆盖方案 §3.6 中与纯函数相关的部分（识别 + 锁定公式），
// 跨局/换边/权限表派生的用例见 game_service_test.go。
// ============================================================================

// TestYuanliu_IsYuanliuHero 前缀识别：必须命中五个元流，且不误伤其它英雄。
func TestYuanliu_IsYuanliuHero(t *testing.T) {
	tests := []struct {
		name string
		hero string
		want bool
	}{
		// --- 五个元流：分路各不相同，名字前缀一致 ---
		{"元流坦克", "元流之子·坦克", true},
		{"元流法师", "元流之子·法师", true},
		{"元流射手", "元流之子·射手", true},
		{"元流刺客", "元流之子·刺客", true},
		{"元流辅助", "元流之子·辅助", true},

		// --- ⚠️ 防"元"字误伤：这些都不是元流之子 ---
		{"元歌是独立英雄", "元歌", false},
		{"曜不是元流", "曜", false},
		{"瑶不是元流", "瑶", false},
		{"元流无之子后缀", "元流", false},
		{"元流之子带空格", "  元流之子·坦克  ", true}, // TrimSpace 后命中

		// --- 边界 ---
		{"空名字", "", false},
		{"纯空白", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsYuanliuHero(tt.hero); got != tt.want {
				t.Errorf("IsYuanliuHero(%q) = %v, 期望 %v", tt.hero, got, tt.want)
			}
		})
	}
}

// TestYuanliu_GroupIDs 从英雄列表筛出元流组 ID。
func TestYuanliu_GroupIDs(t *testing.T) {
	heroes := []*model.Hero{
		{ID: 1, Name: "关羽"},
		{ID: 33, Name: "元流之子·坦克"},
		{ID: 60, Name: "元流之子·法师"},
		{ID: 61, Name: "元歌"}, // 干扰项：应以"元"开头但不是元流
		{ID: 83, Name: "元流之子·射手"},
		{ID: 105, Name: "元流之子·刺客"},
		{ID: 132, Name: "元流之子·辅助"},
		{ID: 133, Name: "王维"},
	}

	got := YuanliuGroupIDs(heroes)
	want := []int{33, 60, 83, 105, 132}

	if len(got) != len(want) {
		t.Fatalf("元流组应有 %d 个，实际 %d 个：%v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第%d个元流 ID = %d, 期望 %d（顺序应稳定）", i, got[i], want[i])
		}
	}
}

// TestYuanliu_GroupIDs_Empty 英雄池无元流英雄时返回空切片，不 panic。
func TestYuanliu_GroupIDs_Empty(t *testing.T) {
	heroes := []*model.Hero{{ID: 1, Name: "关羽"}, {ID: 2, Name: "元歌"}}

	got := YuanliuGroupIDs(heroes)
	if len(got) != 0 {
		t.Errorf("无元流英雄时应返回空，实际 = %v", got)
	}
}

// TestYuanliu_GroupIDs_NilAndZeroID nil 元素与非法 ID 必须被安全跳过。
func TestYuanliu_GroupIDs_NilAndZeroID(t *testing.T) {
	heroes := []*model.Hero{
		{ID: 0, Name: "元流之子·幽灵"}, // ID 非法 → 跳过
		nil,                      // nil 元素 → 跳过（不 panic）
		{ID: 33, Name: "元流之子·坦克"},
	}

	got := YuanliuGroupIDs(heroes)
	if len(got) != 1 || got[0] != 33 {
		t.Errorf("应只筛出 ID=33，实际 = %v", got)
	}
}

// TestYuanliu_Locked_Empty 未用过任何元流 → 锁定集为空（五选一权力完整）。
//
// 这对应需求 1 的前提状态：未使用前，五个元流全部可选。
func TestYuanliu_Locked_Empty(t *testing.T) {
	group := []int{33, 60, 83, 105, 132}

	got := YuanliuLocked(group, map[int]bool{})
	if len(got) != 0 {
		t.Errorf("未用过元流时不应有锁定，实际 = %v", got)
	}

	// nil map 同样安全（等价于"未用过"）
	if got := YuanliuLocked(group, nil); len(got) != 0 {
		t.Errorf("nil usedByTeam 不应有锁定，实际 = %v", got)
	}
}

// TestYuanliu_Locked_OneUsed 已 pick 元射(83) → 锁定其余四个，**元射自身不在锁定集**。
//
// 这是需求 1 的直译：「一旦使用了元射，则其它四个元流都不能再选」。
// 元射自身不在锁定集内，是因为它已从英雄池移除、本局天然不可再选；
// 对它的"不可再选"由全局BP记忆规则负责（自己选过 → 不可再选）。
func TestYuanliu_Locked_OneUsed(t *testing.T) {
	group := []int{33, 60, 83, 105, 132}
	want := map[int]bool{33: true, 60: true, 105: true, 132: true}

	got := YuanliuLocked(group, map[int]bool{83: true})

	if len(got) != len(want) {
		t.Fatalf("应锁定 %d 个，实际 %d 个：%v", len(want), len(got), got)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("英雄%d应被锁定", id)
		}
	}
	if got[83] {
		t.Error("元射(83)自身不应在锁定集内（它由英雄池移除 + 全局BP记忆处理）")
	}
}

// TestYuanliu_Locked_NonShooterUsed 用"坦克"举例 —— 证明规则对五个元流一视同仁。
//
// 需求原文以元射举例，但规则必须对称：用过任何一个，其余四个都被锁。
func TestYuanliu_Locked_NonShooterUsed(t *testing.T) {
	group := []int{33, 60, 83, 105, 132}
	want := map[int]bool{60: true, 83: true, 105: true, 132: true}

	got := YuanliuLocked(group, map[int]bool{33: true})

	if len(got) != len(want) {
		t.Fatalf("应锁定 %d 个，实际 %d 个：%v", len(want), len(got), got)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("英雄%d应被锁定", id)
		}
	}
	if got[33] {
		t.Error("坦克(33)自身不应在锁定集内")
	}
}

// TestYuanliu_Locked_AllUsed 极端情形：五个都用过（理论不可达）→ 锁定集为空。
func TestYuanliu_Locked_AllUsed(t *testing.T) {
	group := []int{33, 60, 83, 105, 132}
	used := map[int]bool{33: true, 60: true, 83: true, 105: true, 132: true}

	got := YuanliuLocked(group, used)
	if len(got) != 0 {
		t.Errorf("五个都用过时锁定集应为空（无剩余可锁），实际 = %v", got)
	}
}

// TestYuanliu_Locked_IgnoresNonGroupUsage used 中存在非元流英雄时不影响本组锁定。
func TestYuanliu_Locked_IgnoresNonGroupUsage(t *testing.T) {
	group := []int{33, 60, 83, 105, 132}
	// 用过元射 + 一堆非元流英雄
	used := map[int]bool{83: true, 1: true, 61: true}

	got := YuanliuLocked(group, used)

	// 结果应与"只用过元射"完全一致
	want := map[int]bool{33: true, 60: true, 105: true, 132: true}
	if len(got) != len(want) {
		t.Fatalf("锁定集应与只用过元射时一致，实际 = %v", got)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("英雄%d应被锁定", id)
		}
	}
}

// TestYuanliu_Locked_EmptyGroup 元流组为空（英雄池中无元流）→ 不 panic，空集。
func TestYuanliu_Locked_EmptyGroup(t *testing.T) {
	got := YuanliuLocked(nil, map[int]bool{83: true})
	if len(got) != 0 {
		t.Errorf("空组不应产生锁定，实际 = %v", got)
	}
}

// TestYuanliu_Locked_DoesNotAssumeFive 代码不假设元流恰好 5 个 —— 未来新增第六个时规则自动覆盖。
func TestYuanliu_Locked_DoesNotAssumeFive(t *testing.T) {
	group := []int{33, 60, 83, 105, 132, 200} // 假设官方新增了第六个元流
	used := map[int]bool{83: true}

	got := YuanliuLocked(group, used)

	// 其余五个（含新增的 200）都应被锁
	want := map[int]bool{33: true, 60: true, 105: true, 132: true, 200: true}
	if len(got) != len(want) {
		t.Fatalf("应锁定 %d 个，实际 %d 个：%v", len(want), len(got), got)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("英雄%d应被锁定", id)
		}
	}
}
