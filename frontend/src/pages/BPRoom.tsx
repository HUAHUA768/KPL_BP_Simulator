import React, { useEffect, useState, useMemo } from 'react'
import { useParams, useSearchParams, useNavigate } from 'react-router-dom'
import BanPickBoard from '../components/BanPickBoard'
import { useWebSocket } from '../hooks/useWebSocket'
import { useBPState } from '../hooks/useBPState'
import {
  getRoomStatus,
  getHeroes,
  getTeams,
  banHero,
  pickHero,
  submitGameResult,
  nextGame,
  finishMatch,
} from '../services/api'
import { useBPStore } from '../store/bpStore'
import { parseMode, resolveActor } from '../utils/actor'
import type { Hero } from '../types'

// 分路常量
const LANES = ['全部', '对抗路', '中路', '发育路', '打野', '游走'] as const
type Lane = (typeof LANES)[number]

// 赛制显示：0 表示无限制，其余 BO{n}
const boLabel = (bo: number) => (bo > 0 ? `BO${bo}` : '无限制')

// 夺冠所需胜场（与后端 Room.winsNeeded 一致；0=永不自动结束）
const winsNeeded = (bo: number) => (bo > 0 ? Math.floor(bo / 2) + 1 : 0)

const BPRoom: React.FC = () => {
  const { id: roomId } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const mySide = searchParams.get('side') || 'blue'

  const bpState = useBPState()
  const setHeroList = useBPStore((s) => s.setHeroList)
  const updateBPState = useBPStore((s) => s.updateBPState)
  const setMode = useBPStore((s) => s.setMode)

  // 交互模式：solo（默认，一人分饰两角）/ dual（双人预留）
  const mode = parseMode(searchParams.toString())
  // 有效操作方：solo = 引擎当前回合方；dual = URL 声明方
  const actorSide = resolveActor(bpState.side, mode, mySide)

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedLane, setSelectedLane] = useState<Lane>('全部')
  const [submitting, setSubmitting] = useState(false)
  // 队名映射（id → name），用于结算面板显示真实队名
  const [teamNames, setTeamNames] = useState<Record<number, string>>({})
  // 队标映射（id → logoUrl），用于局间选边 UI 展示
  const [teamLogos, setTeamLogos] = useState<Record<number, string>>({})

  // 下一局选边（问题4）。null = 沿用当前红蓝归属（默认不换，用户可改）
  const [nextBlueId, setNextBlueId] = useState<number | null>(null)
  const [nextRedId, setNextRedId] = useState<number | null>(null)

  const blueTeamName =
    (bpState.blueTeamId != null && teamNames[bpState.blueTeamId]) || '蓝方队伍'
  const redTeamName =
    (bpState.redTeamId != null && teamNames[bpState.redTeamId]) || '红方队伍'

  // 根据选中的分路过滤英雄
  const filteredHeroes = useMemo(() => {
    if (selectedLane === '全部') return bpState.heroList
    return bpState.heroList.filter((hero) => hero.lanes.includes(selectedLane))
  }, [bpState.heroList, selectedLane])

  // WebSocket 连接（纯下行订阅；写路径唯一走 REST）
  useWebSocket(roomId || '')

  // 初始化：写入交互模式 + 加载英雄列表 + 获取房间状态 + 建立队名映射
  useEffect(() => {
    setMode(mode)
    if (!roomId) return

    Promise.all([getHeroes(), getRoomStatus(roomId), getTeams().catch(() => [])])
      .then(([heroes, state, teams]) => {
        setHeroList(heroes)
        updateBPState(state)
        setTeamNames(Object.fromEntries(teams.map((t) => [t.id, t.name])))
        setTeamLogos(Object.fromEntries(teams.map((t) => [t.id, t.logoUrl])))
        setLoading(false)
      })
      .catch((err) => {
        setError('加载失败: ' + (err?.response?.data?.message || err.message))
        setLoading(false)
      })
  }, [roomId, mode, setMode, setHeroList, updateBPState])

  // 处理 ban 操作（REST 是唯一写路径，响应即权威状态）
  const handleBanHero = async (hero: Hero) => {
    if (!roomId) return
    try {
      const state = await banHero(roomId, hero.id)
      updateBPState(state)
    } catch (err: any) {
      setError(err?.response?.data?.message || '操作失败')
    }
  }

  // 处理 pick 操作（REST 是唯一写路径，响应即权威状态）
  const handlePickHero = async (hero: Hero) => {
    if (!roomId) return
    try {
      const state = await pickHero(roomId, hero.id)
      updateBPState(state)
    } catch (err: any) {
      setError(err?.response?.data?.message || '操作失败')
    }
  }

  // 通用结算动作：提交胜方 / 下一局 / 结束比赛
  const runAction = async (fn: (id: string) => Promise<any>) => {
    if (!roomId || submitting) return
    setSubmitting(true)
    setError(null)
    try {
      updateBPState(await fn(roomId))
    } catch (err: any) {
      setError(err?.response?.data?.message || '操作失败')
    } finally {
      setSubmitting(false)
    }
  }

  // 当前局的结算阶段判定
  const bpFinished = bpState.status === 'finished'
  const isMatchOver = bpState.matchStatus === 'finished'
  const need = winsNeeded(bpState.boFormat)
  const winnerName = bpState.winner === 'blue' ? blueTeamName : redTeamName
  // 冠军队名（比分高的一方；无限制手动结束时可能平局）
  const championName =
    bpState.blueScore > bpState.redScore
      ? blueTeamName
      : bpState.redScore > bpState.blueScore
        ? redTeamName
        : null

  // 每局开始时清空选边意图（问题4）。
  // 否则第 3 局会沿用第 2 局残留的选择，"默认不换"语义会错乱。
  useEffect(() => {
    setNextBlueId(null)
    setNextRedId(null)
  }, [bpState.gameNumber])

  // 下一局的有效红蓝归属：未选择时 = 沿用当前局（默认不换）
  const effNextBlueId = nextBlueId ?? bpState.blueTeamId
  const effNextRedId = nextRedId ?? bpState.redTeamId
  const sidesChanged =
    nextBlueId !== null && nextRedId !== null && nextBlueId !== bpState.blueTeamId

  const nameOf = (id: number | null | undefined) =>
    (id != null && teamNames[id]) || '未知队伍'
  const logoOf = (id: number | null | undefined) =>
    id != null ? teamLogos[id] : undefined

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-white text-xl">加载中...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-900 p-4">
      {/* 顶部导航 */}
      <div className="max-w-7xl mx-auto">
        <div className="mb-4 flex flex-wrap items-center justify-center gap-x-4 gap-y-2">
          <button
            className="rounded-md border border-gray-700 bg-gray-800/60 px-3 py-1.5 text-sm text-gray-300 transition-colors hover:border-gray-500 hover:text-white"
            onClick={() => navigate('/')}
          >
            ← 返回首页
          </button>
          <div className="text-center text-sm text-gray-400">
            房间: <span className="text-yellow-400">{roomId}</span>
            <span className="mx-2 text-gray-600">|</span>
            <span className="text-gray-200">
              第 {bpState.gameNumber} 局 · 蓝 {bpState.blueScore} : {bpState.redScore} 红 ·{' '}
              {boLabel(bpState.boFormat)}
            </span>
            <span className="mx-2 text-gray-600">|</span>
            {mode === 'solo' ? (
              <>
                当前扮演:
                <span className={actorSide === 'blue' ? 'text-blue-400' : 'text-red-400'}>
                  {actorSide === 'blue' ? ' 🔵 蓝方' : ' 🔴 红方'}
                </span>
              </>
            ) : (
              <>
                你:
                <span className={mySide === 'blue' ? 'text-blue-400' : 'text-red-400'}>
                  {mySide === 'blue' ? ' 🔵 蓝方' : ' 🔴 红方'}
                </span>
              </>
            )}
          </div>
        </div>

        {/* 错误提示 */}
        {error && (
          <div className="bg-red-900/40 border border-red-500 text-red-300 rounded px-4 py-2 mb-4 text-center">
            {error}
            <button
              className="ml-2 text-red-400 hover:text-red-200 underline"
              onClick={() => setError(null)}
            >
              关闭
            </button>
          </div>
        )}

        {/* 局间结算面板：录入胜负 → 下一局 / 结束比赛 */}
        {bpFinished && (
          <div className="mb-4 rounded-lg border border-green-500/50 bg-green-900/20 p-4 text-center">
            {isMatchOver ? (
              /* ③ 比赛结束 */
              <>
                <div className="text-lg font-bold text-yellow-300">
                  🏆 比赛结束！
                  {championName
                    ? `${championName} 以 ${Math.max(bpState.blueScore, bpState.redScore)}:${Math.min(bpState.blueScore, bpState.redScore)} 赢得系列赛`
                    : `双方 ${bpState.blueScore}:${bpState.redScore} 战平`}
                </div>
                <div className="mt-2 text-sm text-gray-300">
                  🔵 {blueTeamName} {bpState.blueScore} : {bpState.redScore} {redTeamName} 🔴 ·{' '}
                  {boLabel(bpState.boFormat)}
                </div>
                <div className="mt-3 flex flex-wrap items-center justify-center gap-3">
                  <button
                    className="rounded-md bg-yellow-500 px-4 py-2 text-sm font-bold text-gray-900 transition-colors hover:bg-yellow-400"
                    onClick={() => navigate('/')}
                  >
                    返回首页
                  </button>
                  <button
                    className="rounded-md border border-gray-500 px-4 py-2 text-sm font-bold text-gray-200 transition-colors hover:border-gray-300 hover:text-white"
                    onClick={() => navigate('/history')}
                  >
                    查看历史记录
                  </button>
                </div>
              </>
            ) : bpState.winner === null ? (
              /* ① 本局 BP 完成，等待录入胜负 */
              <>
                <div className="text-lg font-bold text-green-300">
                  🎉 第 {bpState.gameNumber} 局 BP 完成！双方阵容已确定
                </div>
                <div className="mt-2 text-sm text-gray-300">请选择本局胜方：</div>
                <div className="mt-3 flex flex-wrap items-center justify-center gap-3">
                  <button
                    disabled={submitting}
                    className="rounded-md bg-blue-600 px-5 py-2 text-sm font-bold text-white transition-colors hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
                    onClick={() => runAction((id) => submitGameResult(id, 'blue'))}
                  >
                    🔵 {blueTeamName} 获胜
                  </button>
                  <button
                    disabled={submitting}
                    className="rounded-md bg-red-600 px-5 py-2 text-sm font-bold text-white transition-colors hover:bg-red-500 disabled:cursor-not-allowed disabled:opacity-50"
                    onClick={() => runAction((id) => submitGameResult(id, 'red'))}
                  >
                    🔴 {redTeamName} 获胜
                  </button>
                </div>
              </>
            ) : (
              /* ② 本局已录入胜负，比赛未结束 */
              <>
                <div className="text-lg font-bold text-green-300">
                  ✅ 第 {bpState.gameNumber} 局结束：{winnerName} 获胜
                </div>
                <div className="mt-2 text-base font-bold text-gray-200">
                  大比分 蓝 {bpState.blueScore} : {bpState.redScore} 红
                  <span className="ml-2 text-sm font-normal text-gray-400">
                    （{boLabel(bpState.boFormat)}
                    {need > 0 ? ` · 先胜 ${need} 局者夺冠` : ' · 不自动结束'}）
                  </span>
                </div>
                <div className="mt-2 text-xs text-gray-400">
                  提示：下一局将启用全局BP规则——本局双方选用过的英雄，下一局各自不可再选；
                  对手选用过的英雄仍可选（不可 ban）
                </div>

                {/* 第 N+1 局选边（问题4）：单机模式下由用户决定谁蓝谁红 */}
                <div className="mt-4 rounded-lg border border-gray-600/60 bg-gray-900/40 p-3 text-left">
                  <div className="mb-2 text-sm font-bold text-gray-200">
                    🔄 第 {bpState.gameNumber + 1} 局选边
                  </div>
                  <div className="mb-3 flex flex-wrap items-center justify-center gap-2 text-xs text-gray-400">
                    <span>当前：</span>
                    {logoOf(bpState.blueTeamId) && (
                      <img
                        src={logoOf(bpState.blueTeamId)}
                        alt={blueTeamName}
                        className="h-5 w-5 rounded object-contain"
                      />
                    )}
                    <span className="text-blue-400">🔵 {blueTeamName}</span>
                    <span className="text-gray-500">vs</span>
                    <span className="text-red-400">🔴 {redTeamName}</span>
                    {logoOf(bpState.redTeamId) && (
                      <img
                        src={logoOf(bpState.redTeamId)}
                        alt={redTeamName}
                        className="h-5 w-5 rounded object-contain"
                      />
                    )}
                    <span className="w-full text-center">
                      单机模式下由你决定下一局红蓝归属；联机模式将由败方选边。
                    </span>
                  </div>

                  <div className="flex flex-wrap items-center justify-center gap-3">
                    {/* 蓝方选择 */}
                    <label className="flex items-center gap-2 text-sm">
                      <span className="text-blue-400">🔵 蓝方</span>
                      <select
                        className="rounded-md border border-gray-600 bg-gray-900 px-2 py-1 text-white focus:border-blue-500 focus:outline-none"
                        value={effNextBlueId ?? ''}
                        onChange={(e) => {
                          const id = Number(e.target.value)
                          setNextBlueId(id)
                          // 自动把另一队放到红方，杜绝"两队相同"
                          setNextRedId(id === bpState.blueTeamId ? bpState.redTeamId : bpState.blueTeamId)
                        }}
                      >
                        {[bpState.blueTeamId, bpState.redTeamId]
                          .filter((v): v is number => v != null)
                          .map((id) => (
                            <option key={`nb-${id}`} value={id}>
                              {nameOf(id)}
                            </option>
                          ))}
                      </select>
                    </label>

                    <span className="text-gray-500">VS</span>

                    {/* 红方选择 */}
                    <label className="flex items-center gap-2 text-sm">
                      <span className="text-red-400">🔴 红方</span>
                      <select
                        className="rounded-md border border-gray-600 bg-gray-900 px-2 py-1 text-white focus:border-red-500 focus:outline-none"
                        value={effNextRedId ?? ''}
                        onChange={(e) => {
                          const id = Number(e.target.value)
                          setNextRedId(id)
                          setNextBlueId(id === bpState.blueTeamId ? bpState.redTeamId : bpState.blueTeamId)
                        }}
                      >
                        {[bpState.blueTeamId, bpState.redTeamId]
                          .filter((v): v is number => v != null)
                          .map((id) => (
                            <option key={`nr-${id}`} value={id}>
                              {nameOf(id)}
                            </option>
                          ))}
                      </select>
                    </label>

                    {/* 一键交换红蓝 */}
                    <button
                      disabled={submitting}
                      className="rounded-md border border-gray-500 px-3 py-1 text-xs font-bold text-gray-200 transition-colors hover:border-gray-300 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                      onClick={() => {
                        setNextBlueId(bpState.redTeamId)
                        setNextRedId(bpState.blueTeamId)
                      }}
                    >
                      ⇄ 交换红蓝
                    </button>
                  </div>

                  <div className="mt-2 text-center text-xs text-gray-500">
                    {sidesChanged
                      ? `下一局将交换红蓝：🔵 ${nameOf(effNextRedId)} vs 🔴 ${nameOf(effNextBlueId)}`
                      : '下一局沿用当前红蓝归属（可点「⇄ 交换红蓝」调整）'}
                  </div>
                </div>

                <div className="mt-3 flex flex-wrap items-center justify-center gap-3">
                  <button
                    disabled={submitting}
                    className="rounded-md bg-yellow-500 px-4 py-2 text-sm font-bold text-gray-900 transition-colors hover:bg-yellow-400 disabled:cursor-not-allowed disabled:opacity-50"
                    onClick={() =>
                      runAction((id) =>
                        nextGame(
                          id,
                          effNextBlueId != null && effNextRedId != null
                            ? { blueTeamId: effNextBlueId, redTeamId: effNextRedId }
                            : undefined,
                        ),
                      )
                    }
                  >
                    ➡️ 开始第 {bpState.gameNumber + 1} 局 BP
                  </button>
                  {/* 无限制赛制才提供手动结束 */}
                  {bpState.boFormat === 0 && (
                    <button
                      disabled={submitting}
                      className="rounded-md border border-gray-500 px-4 py-2 text-sm font-bold text-gray-200 transition-colors hover:border-gray-300 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                      onClick={() => runAction(finishMatch)}
                    >
                      结束比赛
                    </button>
                  )}
                </div>
              </>
            )}
          </div>
        )}

        {/* 分路筛选栏（居中） */}
        <div className="mb-4 flex flex-wrap items-center justify-center gap-2">
          <span className="mr-1 text-sm text-gray-400">分路:</span>
          {LANES.map((lane) => (
            <button
              key={lane}
              className={`rounded-full px-4 py-1.5 text-center text-sm font-medium transition-colors duration-200 ${
                selectedLane === lane
                  ? 'bg-yellow-500 text-gray-900'
                  : 'bg-gray-700/60 text-gray-300 hover:bg-gray-600 hover:text-white'
              }`}
              onClick={() => setSelectedLane(lane)}
            >
              {lane}
            </button>
          ))}
        </div>

        {/* BP 棋盘 */}
        <BanPickBoard
          heroes={filteredHeroes}
          allHeroes={bpState.heroList}
          blueBanned={bpState.blueBanned}
          redBanned={bpState.redBanned}
          bluePicked={bpState.bluePicked}
          redPicked={bpState.redPicked}
          currentTurn={bpState.currentTurn}
          action={bpState.action}
          side={bpState.side}
          onBanHero={handleBanHero}
          onPickHero={handlePickHero}
          actorSide={actorSide}
          blueTeam={bpState.blueTeam}
          redTeam={bpState.redTeam}
        />
      </div>
    </div>
  )
}

export default BPRoom