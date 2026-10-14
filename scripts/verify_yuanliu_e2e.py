#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
元流之子「五选一」锁定规则 — 端到端验收脚本

对应施工方案 §4.4（十步场景）与 §4.5（权限位直读）。
直接打真实后端 API + 真实 MySQL 数据，不依赖任何 mock。

用法：python3 scripts/verify_yuanliu_e2e.py
"""

import json
import sys
import urllib.request
import urllib.error

BASE = "http://localhost:8080/api"

# 元流五兄弟（真实库 ID）
YL = {33: "坦克", 60: "法师", 83: "射手", 105: "刺客", 132: "辅助"}
YL_IDS = sorted(YL.keys())
SHOOTER = 83  # 元射

# 20 步状态机（与 bp_engine.go getTurnInfo 一致）
TURNS = [
    ("blue", "ban"), ("red", "ban"), ("blue", "ban"), ("red", "ban"),
    ("blue", "pick"), ("red", "pick"), ("red", "pick"), ("blue", "pick"),
    ("blue", "pick"), ("red", "pick"),
    ("red", "ban"), ("blue", "ban"), ("red", "ban"), ("blue", "ban"),
    ("red", "ban"), ("blue", "ban"),
    ("red", "pick"), ("blue", "pick"), ("blue", "pick"), ("red", "pick"),
]

PASS, FAIL = [], []


def check(name, cond, detail=""):
    (PASS if cond else FAIL).append(name)
    print(f"  {'✅' if cond else '❌'} {name}" + (f"  — {detail}" if detail else ""))


def req(method, path, body=None):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(url, data=data, method=method)
    if data:
        r.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(r, timeout=15) as resp:
            return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode())
    except Exception as e:
        return 0, {"message": str(e)}


def create_room(blue=1, red=2, bo=5):
    st, body = req("POST", "/rooms", {"blueTeamId": blue, "redTeamId": red, "boFormat": bo})
    assert st == 201, f"创建房间失败: {st} {body}"
    return body["data"]["roomId"]


def state(room):
    st, body = req("GET", f"/rooms/{room}/status")
    assert st == 200, f"取状态失败: {st} {body}"
    return body["data"]


def perms_of(s):
    """返回 {heroId: (blueCanBan, blueCanPick, redCanBan, redCanPick)}"""
    out = {}
    for h in s["heroes"]:
        p = h["permissions"]
        out[h["id"]] = (p["blueCanBan"], p["blueCanPick"], p["redCanBan"], p["redCanPick"])
    return out


def act(room, action, hero):
    """执行一步；返回 (status, body)"""
    return req("POST", f"/rooms/{room}/{action}", {"heroId": hero})


# 安全英雄池：真实库中 1..133，排除五个元流（33/60/83/105/132）。
# 用显式列表而非区间运算，避免"某步算出的 ID 恰好是元流"这类隐蔽冲突。
SAFE_HEROES = [i for i in range(1, 134) if i not in YL_IDS]


def default_hero(step):
    """按步序取一个安全英雄（ban/pick 统一从安全池顺序取，保证互不重复）。"""
    return SAFE_HEROES[(step - 1) % len(SAFE_HEROES)]


def play_until(room, target_step, custom):
    """
    从当前步走到 target_step（1-based）。custom = {step: heroId} 覆盖默认选人。
    默认策略：从 SAFE_HEROES 顺序取（已排除五个元流 ID，不会误触互斥组）。
    """
    s = state(room)
    cur = s["currentTurn"]
    for step in range(cur, target_step + 1):
        side, action = TURNS[step - 1]
        hero = custom.get(step)
        if hero is None:
            hero = default_hero(step)
        st, body = act(room, action, hero)
        assert st == 200, f"第{step}步({action},h{hero})失败: {st} {body}"
    return state(room)


def run_full_game(room, custom):
    """跑完当前局剩余步骤"""
    s = state(room)
    for step in range(s["currentTurn"], 21):
        side, action = TURNS[step - 1]
        hero = custom.get(step)
        if hero is None:
            hero = default_hero(step)
        st, body = act(room, action, hero)
        assert st == 200, f"第{step}步({action},h{hero})失败: {st} {body}"
    return state(room)


def dump_yl(s, label):
    print(f"\n  【{label}】")
    pr = perms_of(s)
    print(f"    {'英雄':<6} {'蓝ban':<7}{'蓝pick':<8}{'红ban':<7}{'红pick':<8}")
    for hid in YL_IDS:
        bb, bp, rb, rp = pr[hid]
        print(f"    {YL[hid]:<6} {str(bb):<7}{str(bp):<8}{str(rb):<7}{str(rp):<8}")
    return pr


# ============================================================================
# 场景 A：同局 —— 蓝方 pick 元射后（需求 1、2、3）
# ============================================================================
def scenario_a():
    print("\n" + "=" * 72)
    print("场景 A：同局内 蓝方 pick 元射 后的权限变化（需求 1 / 2 / 3）")
    print("=" * 72)

    room = create_room()
    print(f"\n房间: {room}")

    # 步骤 2：BP 开始，五个元流全部可 ban 可 pick
    s = state(room)
    pr = dump_yl(s, "开局（未使用任何元流）")
    check("A1 开局：蓝方对五个元流全可pick（五选一权力完整）",
          all(pr[i][1] for i in YL_IDS))
    check("A2 开局：蓝方对五个元流全可ban", all(pr[i][0] for i in YL_IDS))
    check("A3 开局：红方对五个元流全可pick", all(pr[i][3] for i in YL_IDS))

    # 走到第5步（蓝pick1）并 pick 元射
    play_until(room, 4, {})
    st, _ = act(room, "pick", SHOOTER)
    check("A4 蓝方成功 pick 元射", st == 200, f"HTTP {st}")
    s = state(room)
    pr = dump_yl(s, "蓝方已 pick 元射（本局第5步）")

    others = [i for i in YL_IDS if i != SHOOTER]

    # 需求 1：其余四个不可再 pick
    check("A5【需求1】蓝方对其余四个元流不可pick",
          all(not pr[i][1] for i in others),
          "、".join(YL[i] for i in others if pr[i][1]) or "全部已锁")

    # 需求 2：蓝方 ban 权完整保留
    check("A6【需求2】蓝方对五个元流全部保有 ban 权",
          all(pr[i][0] for i in YL_IDS))

    # 需求 2：红方仍可五选一
    check("A7【需求2】红方对其余四个元流仍可pick（B队依然五选一）",
          all(pr[i][3] for i in others))

    # 需求 2（用户实测报告 1）："若 A 队已使用元射，则 B 队不用再 ban 其它四个元流"
    # → 狼队（红方）对这些元流**不应有 ban 权**
    check("A7b【需求2·报告1】红方对其余四个元流不再可ban（B队不用再ban）",
          all(not pr[i][2] for i in others),
          "、".join(YL[i] for i in others if pr[i][2]) or "全部已禁止ban")

    # 对照：AG（蓝方）自身 ban 权必须保留（需求 2 的另一半）
    check("A7c【需求2】蓝方自身对五个元流仍全可ban（对照，未误伤）",
          all(pr[i][0] for i in YL_IDS))

    # 需求 3：后端二次校验必须独立拒绝非法 pick。
    # ⚠️ 必须等**轮到蓝方 pick** 时再测 —— 第5步之后第6/7步是红方 pick，
    #    若此时提交会被"动作类型不匹配"拒绝（那是另一个错误码），测不到本规则。
    #    蓝方下一次 pick 是第 8 步（蓝pick2），故先走到第 7 步结束。
    play_until(room, 7, {})
    s = state(room)
    check("A8 前置：当前轮到蓝方 pick",
          s["side"] == "blue" and s["action"] == "pick",
          f"side={s['side']} action={s['action']} turn={s['currentTurn']}")

    st, body = act(room, "pick", 33)
    check("A9【需求3兜底】蓝方再 pick 坦克被后端拒绝",
          st == 400 and body.get("code") == 40001,
          f"HTTP {st} code={body.get('code')} msg={body.get('message')}")

    # 确认拒绝原因**确实是元流锁定**（而非"已被ban/pick占用"等其它原因）：
    # 坦克此刻既未被 ban 也未被 pick，唯一的原因只可能是元流五选一锁定。
    check("A10 拒绝原因确为「受全局BP规则限制」（而非占用类原因）",
          "受全局BP规则限制" in str(body.get("message", "")),
          f"msg={body.get('message')}")

    return room


# ============================================================================
# 场景 B：同局 —— ban 轮次 A 队仍能 ban 其余四个（需求 2 核心）
# ============================================================================
def scenario_b(room):
    print("\n" + "=" * 72)
    print("场景 B：同局 ban 轮次 —— A 队仍能 ban 其余四个（需求 2 核心）")
    print("=" * 72)

    # 当前在第6步(红pick1)。走到第11步结束（即停在第12步=蓝ban4）
    play_until(room, 11, {})
    s = state(room)
    check("B0 当前轮到蓝方 ban", s["side"] == "blue" and s["action"] == "ban",
          f"side={s['side']} action={s['action']}")

    st, body = act(room, "ban", 33)
    check("B1【需求2】蓝方已用过元射，仍能 ban 坦克",
          st == 200, f"HTTP {st} msg={body.get('message','')}")

    s = state(room)
    check("B2 坦克已进入蓝方 ban 列表", 33 in s["blueBanned"])

    # 报告 1 的直接复现：轮到**狼队（红）ban** 时，它不应能 ban 元坦
    # 第13步 = 红ban4（红方在第二循环的 ban 步）
    play_until(room, 12, {})
    s = state(room)
    check("B3 前置：当前轮到红方 ban",
          s["side"] == "red" and s["action"] == "ban",
          f"side={s['side']} action={s['action']} turn={s['currentTurn']}")

    st, body = act(room, "ban", 60)  # 元流之子·法师
    check("B4【报告1复现】红方（狼队）ban 元流·法师被拒绝",
          st == 400 and body.get("code") == 40001,
          f"HTTP {st} code={body.get('code')} msg={body.get('message')}")


# ============================================================================
# 场景 C：跨局累计（需求 1「本场比赛」）
# ============================================================================
def scenario_c(room):
    print("\n" + "=" * 72)
    print("场景 C：跨局累计 —— 第 2 局仍锁定（需求 1「本场比赛」）")
    print("=" * 72)

    # 跑完第1局
    run_full_game(room, {})
    s = state(room)
    check("C1 第1局 BP 已完成", s["status"] == "finished", f"status={s['status']}")

    st, _ = req("POST", f"/rooms/{room}/result", {"winner": "blue"})
    check("C2 提交本局胜方成功", st == 200, f"HTTP {st}")

    # 进入第2局（不换边）
    st, body = req("POST", f"/rooms/{room}/next-game",
                   {"blueTeamId": s["blueTeamId"], "redTeamId": s["redTeamId"]})
    if st != 200:
        st, body = req("POST", f"/rooms/{room}/next-game", None)
    check("C3 进入第2局成功", st == 200, f"HTTP {st} {body.get('message','')}")

    s2 = state(room)
    check("C4 当前是第2局", s2["gameNumber"] == 2, f"gameNumber={s2['gameNumber']}")

    pr = dump_yl(s2, "第2局 未换边（蓝方仍是第1局用过元射的那队）")
    others = [i for i in YL_IDS if i != SHOOTER]

    check("C5【需求1】第2局蓝方对其余四个元流仍不可pick（跨局累计）",
          all(not pr[i][1] for i in others))
    check("C6【需求2】第2局蓝方对五个元流仍全可ban",
          all(pr[i][0] for i in YL_IDS))
    check("C7【需求2】第2局红方对其余四个仍可pick（B队依然五选一）",
          all(pr[i][3] for i in others))
    # 报告 2 的直接复现（跨局路径）：狼队第2局也不能 ban 元坦等四个
    check("C8【报告2复现】第2局红方（狼队）对其余四个元流不可ban",
          all(not pr[i][2] for i in others),
          "、".join(YL[i] for i in others if pr[i][2]) or "全部已禁止ban")

    return s2


# ============================================================================
# 场景 D：换边后限制仍「跟队伍走」（§8.1 回归防护）
# ============================================================================
def scenario_d():
    print("\n" + "=" * 72)
    print("场景 D：第2局换边 —— 限制跟队伍走，不跟颜色走（§8.1 回归防护）")
    print("=" * 72)

    room = create_room()
    s0 = state(room)
    blue0, red0 = s0["blueTeamId"], s0["redTeamId"]

    # 第1局：蓝方 pick 元射
    play_until(room, 4, {})
    st, _ = act(room, "pick", SHOOTER)
    assert st == 200, "前置：蓝方 pick 元射失败"
    run_full_game(room, {})
    st, _ = req("POST", f"/rooms/{room}/result", {"winner": "blue"})
    assert st == 200, "前置：提交胜方失败"

    # 第2局：**换边**（蓝红互换）
    st, body = req("POST", f"/rooms/{room}/next-game",
                   {"blueTeamId": red0, "redTeamId": blue0})
    check("D1 换边进入第2局成功", st == 200, f"HTTP {st} {body.get('message','')}")

    s2 = state(room)
    check("D2 红蓝归属确实已互换",
          s2["blueTeamId"] == red0 and s2["redTeamId"] == blue0,
          f"蓝={s2['blueTeamId']} 红={s2['redTeamId']}（第1局 蓝={blue0} 红={red0}）")

    pr = dump_yl(s2, "第2局 已换边（用过元射的队现在穿红色）")
    others = [i for i in YL_IDS if i != SHOOTER]

    # 用过元射的队现在是红 → 红方对其余四个不可 pick
    check("D3【换边】用过元射的队（现为红）对其余四个不可pick",
          all(not pr[i][3] for i in others))
    # 但它的 ban 权仍完整
    check("D4【换边】该队（现为红）对五个元流仍全可ban",
          all(pr[i][2] for i in YL_IDS))
    # 对手（现为蓝）不受影响
    check("D5【换边】对手（现为蓝）对其余四个仍可pick（限制未加错队）",
          all(pr[i][1] for i in others))
    # 换边后对手（现为蓝的狼队）同样不可 ban
    check("D6【换边】对手（现为蓝）对其余四个不可ban（规则跟队伍走）",
          all(not pr[i][0] for i in others))
    # 用过元流的队（现为红）自身 ban 权仍在
    check("D7【换边】用过元流的队（现为红）自身仍全可ban",
          all(pr[i][2] for i in YL_IDS))


def main():
    print("元流之子「五选一」锁定规则 — 端到端验收")
    print(f"后端: {BASE}")

    try:
        room = scenario_a()
        scenario_b(room)

        # 场景 C 用全新房间，避免 A/B 的 ban 干扰
        room2 = create_room()
        play_until(room2, 4, {})
        st, _ = act(room2, "pick", SHOOTER)
        assert st == 200, "场景C前置：蓝方 pick 元射失败"
        scenario_c(room2)

        scenario_d()
    except AssertionError as e:
        print(f"\n❌ 脚本中断: {e}")
        FAIL.append(f"脚本中断: {e}")

    print("\n" + "=" * 72)
    print(f"结果：通过 {len(PASS)} 项，失败 {len(FAIL)} 项")
    if FAIL:
        print("\n失败项：")
        for f in FAIL:
            print(f"  ❌ {f}")
        sys.exit(1)
    print("✅ 全部通过")
    sys.exit(0)


if __name__ == "__main__":
    main()