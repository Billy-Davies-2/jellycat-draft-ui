'use client'

import { useEffect, useMemo, useState, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Crown, Heart, Star, Trophy, Users, Clock, Sparkles, Medal } from 'lucide-react'
import { trpc } from '@/lib/trpc-client'

interface Player {
  id: string
  name: string
  position: string
  team: string
  points: number
  tier: 'S' | 'A' | 'B' | 'C'
  drafted: boolean
  draftedBy?: string
  image: string
}

interface Team {
  id: string
  name: string
  owner: string
  mascot: string
  color: string
  players: Player[]
}

interface ChatMessage {
  id: string
  ts: number
  type: 'system' | 'user'
  text: string
  emotes: Record<string, number>
}

export default function JellycatFantasyDraft() {
  const [players, setPlayers] = useState<Player[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const [chat, setChat] = useState<ChatMessage[]>([])
  const [currentPick, setCurrentPick] = useState(1)
  const [currentTeam, setCurrentTeam] = useState(0)
  const [owner, setOwner] = useState('')
  const [selectedPlayer, setSelectedPlayer] = useState<Player | null>(null)
  const [profile, setProfile] = useState<any | null>(null)
  const [draftStarted, setDraftStarted] = useState(false)
  const [draftComplete, setDraftComplete] = useState(false)

  const DRAFT_ROUNDS = 3
  // Planned maximum based on rounds
  const plannedTotalPicks = useMemo(() => teams.length * DRAFT_ROUNDS, [teams.length])
  // Actual cap cannot exceed number of players available
  const totalPicks = useMemo(
    () => Math.min(players.length, plannedTotalPicks),
    [players.length, plannedTotalPicks],
  )

  const getTierColor = (tier: string) => {
    switch (tier) {
      case 'S':
        return 'bg-gradient-to-r from-yellow-200 to-yellow-300 text-yellow-800'
      case 'A':
        return 'bg-gradient-to-r from-green-200 to-green-300 text-green-800'
      case 'B':
        return 'bg-gradient-to-r from-blue-200 to-blue-300 text-blue-800'
      case 'C':
        return 'bg-gradient-to-r from-gray-200 to-gray-300 text-gray-800'
      default:
        return 'bg-gray-100'
    }
  }

  const getPositionColor = (position: string) => {
    switch (position) {
      case 'CC':
        return 'bg-pink-100 text-pink-700 border-pink-200'
      case 'SS':
        return 'bg-purple-100 text-purple-700 border-purple-200'
      case 'HH':
        return 'bg-blue-100 text-blue-700 border-blue-200'
      case 'CH':
        return 'bg-green-100 text-green-700 border-green-200'
      default:
        return 'bg-gray-100 text-gray-700 border-gray-200'
    }
  }

  async function loadState() {
    const data = await trpc.draft.state.query()
    setPlayers(data.players)
    setTeams(data.teams)
    try {
      const chatData = await trpc.chat.list.query()
      setChat(Array.isArray(chatData) ? (chatData as ChatMessage[]) : [])
    } catch (e) {
      setChat([])
    }
    return data
  }

  useEffect(() => {
    loadState()
    try {
      const o = localStorage.getItem('jellycat_owner')
      if (o) setOwner(o)
    } catch {}
    // subscribe to server events to refresh on changes
    const sub = (trpc as any).events.subscribe(undefined, {
      onData: () => loadState(),
      onError: () => {},
    })
    return () => sub.unsubscribe?.()
  }, [])

  // Determine user's team index (dev-mode pseudo identity by owner name)
  const myTeamIndex = useMemo(() => {
    if (!owner) return -1
    return teams.findIndex((t) => t.owner === owner)
  }, [teams, owner])

  // Auto-pick logic for other teams in dev mode
  const autoPickingRef = useRef(false)
  const autoPickTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (process.env.NODE_ENV !== 'development') return
    if (!draftStarted || draftComplete) return
    if (teams.length === 0) return
    if (myTeamIndex === -1) return // no identified user team; skip
    if (currentTeam === myTeamIndex) return // it's user's turn
    if (autoPickingRef.current) return // already scheduled
    const available = players.filter((p) => !p.drafted)
    if (available.length === 0) return
    autoPickingRef.current = true
    autoPickTimerRef.current = setTimeout(async () => {
      try {
        const teamId = teams[currentTeam]?.id
        if (!teamId) return
        const pool = players.filter((p) => !p.drafted)
        if (pool.length === 0) return
        const choice = pool[Math.floor(Math.random() * pool.length)]
        await trpc.draft.pick.mutate({ playerId: choice.id, teamId })
        await loadState()
        // Advance pick pointers similar to manual flow
        const nextPick = currentPick + 1
        const remaining = pool.length - 1
        const maxPicks = Math.min(players.length, teams.length * DRAFT_ROUNDS)
        if (remaining === 0 || nextPick > maxPicks) {
          setDraftComplete(true)
        } else {
          setCurrentPick(nextPick)
          setCurrentTeam((i) => (i + 1) % teams.length)
        }
      } finally {
        autoPickingRef.current = false
      }
    }, 700) // slight delay for UX
    return () => {
      if (autoPickTimerRef.current) clearTimeout(autoPickTimerRef.current)
    }
  }, [draftStarted, draftComplete, teams, currentTeam, players, myTeamIndex, currentPick])

  const draftPlayer = async (player: Player) => {
    if (player.drafted) return
    const teamId = teams[currentTeam]?.id
    if (!teamId) return
    await trpc.draft.pick.mutate({ playerId: player.id, teamId })
    const data = await loadState()
    const remaining = data.players.filter((p) => !p.drafted).length
    const maxPicks = Math.min(data.players.length, data.teams.length * DRAFT_ROUNDS)
    const nextPick = currentPick + 1
    if (remaining === 0 || nextPick > maxPicks) {
      setDraftComplete(true)
    } else {
      setCurrentPick(nextPick)
      setCurrentTeam((i) => (i + 1) % data.teams.length)
    }
    setSelectedPlayer(null)
  }

  const availablePlayers = useMemo(() => players.filter((p) => !p.drafted), [players])

  async function openProfile(p: Player) {
    try {
      const data = await trpc.players.profile.query({ id: p.id })
      setProfile(data)
    } catch {
      setProfile({ error: 'Failed to load profile', id: p.id })
    }
    setSelectedPlayer(p)
  }

  const getTeamStats = (team: Team) => {
    const totalCuddlePoints = team.players.reduce((sum, player) => sum + player.points, 0)
    const averagePoints =
      team.players.length > 0 ? Math.round(totalCuddlePoints / team.players.length) : 0
    const tierCounts = team.players.reduce(
      (acc, player) => {
        acc[player.tier] = (acc[player.tier] || 0) + 1
        return acc
      },
      {} as Record<string, number>,
    )

    return { totalCuddlePoints, averagePoints, tierCounts }
  }

  const getLeaderboard = () => {
    return teams
      .map((team) => ({ ...team, ...getTeamStats(team) }))
      .sort((a, b) => b.totalCuddlePoints - a.totalCuddlePoints)
  }

  // Results Page
  if (draftComplete) {
    const leaderboard = getLeaderboard()
    const winner = leaderboard[0]

    return (
      <div className="min-h-screen bg-gradient-to-br from-pink-50 via-purple-50 to-blue-50">
        <div className="container mx-auto px-4 py-8">
          <div className="text-center mb-12">
            <div className="flex justify-center items-center gap-4 mb-6">
              <div className="text-6xl">🏆</div>
              <h1 className="text-5xl font-bold bg-gradient-to-r from-pink-500 to-purple-600 bg-clip-text text-transparent">
                Draft Complete!
              </h1>
              <div className="text-6xl">🧸</div>
            </div>
            <p className="text-xl text-gray-600 mb-8">
              Congratulations to all teams on building amazing Jellycat collections!
            </p>
          </div>

          <Card className="max-w-2xl mx-auto mb-8 border-4 border-yellow-300 bg-gradient-to-r from-yellow-50 to-orange-50">
            <CardHeader className="bg-gradient-to-r from-yellow-100 to-orange-100">
              <CardTitle className="flex items-center justify-center gap-3 text-2xl">
                <Crown className="text-yellow-500" size={32} />
                <span>🎉 Champion Team 🎉</span>
                <Crown className="text-yellow-500" size={32} />
              </CardTitle>
            </CardHeader>
            <CardContent className="p-6 text-center">
              <div className="text-4xl mb-2">{winner.mascot}</div>
              <div className="text-2xl font-bold mb-2">{winner.name}</div>
              <div className="text-lg text-gray-600 mb-4">Owner: {winner.owner}</div>
              <div className="text-3xl font-bold text-purple-600">
                {winner.totalCuddlePoints} Cuddle Points
              </div>
            </CardContent>
          </Card>

          <Card className="mb-8 border-2 border-pink-200">
            <CardHeader className="bg-gradient-to-r from-pink-100 to-purple-100">
              <CardTitle className="flex items-center justify-center gap-2">
                <Trophy className="text-yellow-500" />
                Final Leaderboard
              </CardTitle>
            </CardHeader>
            <CardContent className="p-6">
              <div className="space-y-4">
                {leaderboard.map((team, index) => (
                  <div
                    key={team.id}
                    className={`flex items-center gap-4 p-4 p-4 rounded-lg border-2 ${team.color}`}
                  >
                    <div className="flex items-center gap-2">
                      {index === 0 && <Medal className="text-yellow-500" size={24} />}
                      {index === 1 && <Medal className="text-gray-400" size={24} />}
                      {index === 2 && <Medal className="text-amber-600" size={24} />}
                      <span className="text-xl font-bold">#{index + 1}</span>
                    </div>
                    <div className="text-2xl">{team.mascot}</div>
                    <div className="flex-1">
                      <div className="font-semibold text-lg">{team.name}</div>
                      <div className="text-sm text-gray-600">Owner: {team.owner}</div>
                    </div>
                    <div className="text-right">
                      <div className="text-xl font-bold text-purple-600">
                        {team.totalCuddlePoints}
                      </div>
                      <div className="text-sm text-gray-500">Cuddle Points</div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
            {teams.map((team) => {
              const stats = getTeamStats(team)
              return (
                <Card
                  key={team.id}
                  className={`border-2 ${team.color.replace('bg-', 'border-').replace('-100', '-300')}`}
                >
                  <CardHeader className={team.color}>
                    <CardTitle className="flex items-center gap-3">
                      <span className="text-2xl">{team.mascot}</span>
                      <div>
                        <div>{team.name}</div>
                        <div className="text-sm font-normal text-gray-600">
                          {stats.totalCuddlePoints} Cuddle Points
                        </div>
                      </div>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="p-4">
                    <div className="space-y-3">
                      {team.players.map((player) => (
                        <div
                          key={player.id}
                          className="flex items-center gap-3 p-3 bg-white rounded-lg border shadow-sm"
                        >
                          <img
                            src={player.image || '/placeholder.svg'}
                            alt={player.name}
                            className="w-12 h-12 rounded-lg object-cover border-2 border-pink-200"
                          />
                          <div className="flex-1">
                            <div className="font-semibold text-sm">{player.name}</div>
                            <div className="text-xs text-gray-600 flex items-center gap-2">
                              <Badge className={`${getPositionColor(player.position)} text-xs`}>
                                {player.position}
                              </Badge>
                              <span>{player.team}</span>
                            </div>
                          </div>
                          <div className="text-sm font-semibold text-purple-600">
                            {player.points}
                          </div>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )
            })}
          </div>

          <div className="text-center">
            <Button
              onClick={() => {
                setDraftComplete(false)
                setDraftStarted(false)
                setCurrentPick(1)
                setCurrentTeam(0)
                loadState()
              }}
              className="bg-gradient-to-r from-pink-500 to-purple-600 hover:from-pink-600 hover:to-purple-700 text-white font-semibold py-3 px-8 rounded-full text-lg"
            >
              <Sparkles className="mr-2" />
              Start New Draft
            </Button>
          </div>
        </div>
      </div>
    )
  }

  if (!draftStarted) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-pink-50 via-purple-50 to-blue-50">
        <div className="container mx-auto px-4 py-8">
          <div className="text-center mb-12">
            <div className="flex justify-center items-center gap-4 mb-6">
              <div className="text-6xl">🧸</div>
              <h1 className="text-5xl font-bold bg-gradient-to-r from-pink-500 to-purple-600 bg-clip-text text-transparent">
                Jellycat Fantasy Draft
              </h1>
              <div className="text-6xl">🏈</div>
            </div>
            <p className="text-xl text-gray-600 mb-8">
              Draft the cuddliest team of Jellycat friends!
            </p>

            <Card className="max-w-2xl mx-auto border-2 border-pink-200 shadow-lg">
              <CardHeader className="bg-gradient-to-r from-pink-100 to-purple-100">
                <CardTitle className="flex items-center justify-center gap-2">
                  <Crown className="text-yellow-500" />
                  Draft Lobby
                  <Crown className="text-yellow-500" />
                </CardTitle>
              </CardHeader>
              <CardContent className="p-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                  {teams.map((team) => (
                    <div key={team.id} className={`p-4 rounded-lg border-2 ${team.color}`}>
                      <div className="flex items-center gap-3">
                        <div className="text-2xl">{team.mascot}</div>
                        <div>
                          <div className="font-semibold">{team.name}</div>
                          <div className="text-sm text-gray-600">Owner: {team.owner}</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>

                <div className="mb-6 p-4 bg-blue-50 rounded-lg border border-blue-200">
                  <div className="text-center">
                    <div className="font-semibold text-blue-800 mb-2">Draft Format</div>
                    <div className="text-sm text-blue-600">
                      Each team will draft up to {DRAFT_ROUNDS} Jellycats • {totalPicks} total picks
                      (capped by available players)
                    </div>
                  </div>
                </div>

                <Button
                  onClick={() => setDraftStarted(true)}
                  className="w-full bg-gradient-to-r from-pink-500 to-purple-600 hover:from-pink-600 hover:to-purple-700 text-white font-semibold py-3 rounded-full text-lg"
                >
                  <Trophy className="mr-2" />
                  Start Draft!
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-pink-50 via-purple-50 to-blue-50">
      <div className="bg-white border-b-2 border-pink-200 shadow-sm">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="text-3xl">🧸</div>
              <h1 className="text-2xl font-bold bg-gradient-to-r from-pink-500 to-purple-600 bg-clip-text text-transparent">
                Jellycat Fantasy Draft
              </h1>
            </div>

            <div className="flex items-center gap-6">
              <div className="flex items-center gap-2">
                <Clock className="text-pink-500" size={20} />
                <span className="font-semibold">
                  Pick #{currentPick} of {totalPicks}
                </span>
              </div>

              <div className={`px-4 py-2 rounded-full border-2 ${teams[currentTeam]?.color ?? ''}`}>
                <div className="flex items-center gap-2">
                  <span className="text-lg">{teams[currentTeam]?.mascot}</span>
                  <span className="font-semibold">{teams[currentTeam]?.name}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-6">
        <Tabs defaultValue="draft" className="w-full">
          <TabsList className="grid w-full grid-cols-3 mb-6 bg-white border-2 border-pink-200">
            <TabsTrigger value="draft" className="data-[state=active]:bg-pink-100">
              <Users className="mr-2" size={16} />
              Draft Board
            </TabsTrigger>
            <TabsTrigger value="teams" className="data-[state=active]:bg-purple-100">
              <Trophy className="mr-2" size={16} />
              Team Rosters
            </TabsTrigger>
            <TabsTrigger value="chat" className="data-[state=active]:bg-blue-100">
              Live Chat
            </TabsTrigger>
          </TabsList>

          <TabsContent value="draft">
            <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
              <div className="lg:col-span-3">
                <Card className="border-2 border-pink-200">
                  <CardHeader className="bg-gradient-to-r from-pink-100 to-purple-100">
                    <CardTitle className="flex items-center gap-2">
                      <Star className="text-yellow-500" />
                      Available Jellycats ({availablePlayers.length})
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="p-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                      {availablePlayers.map((player) => (
                        <Card
                          key={player.id}
                          className={`cursor-pointer transition-all hover:scale-105 border-2 ${
                            selectedPlayer?.id === player.id
                              ? 'border-pink-400 shadow-lg'
                              : 'border-gray-200 hover:border-pink-300'
                          }`}
                          onClick={() => openProfile(player)}
                        >
                          <CardContent className="p-4">
                            <div className="flex flex-col items-center gap-3 mb-3">
                              <img
                                src={player.image || '/placeholder.svg'}
                                alt={player.name}
                                className="w-20 h-20 rounded-xl object-cover border-2 border-pink-200 shadow-sm"
                              />
                              <div className="text-center">
                                <div className="font-semibold text-sm">{player.name}</div>
                                <div className="text-xs text-gray-600">{player.team}</div>
                              </div>
                            </div>

                            <div className="flex items-center justify-between mb-2">
                              <Badge className={`${getPositionColor(player.position)} border`}>
                                {player.position}
                              </Badge>
                              <Badge className={getTierColor(player.tier)}>
                                Tier {player.tier}
                              </Badge>
                            </div>

                            <div className="text-center">
                              <div className="text-lg font-bold text-purple-600">
                                {player.points}
                              </div>
                              <div className="text-xs text-gray-500">Cuddle Points</div>
                            </div>
                          </CardContent>
                        </Card>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              </div>

              <div className="lg:col-span-1">
                <Card className="border-2 border-purple-200 sticky top-4">
                  <CardHeader className="bg-gradient-to-r from-purple-100 to-pink-100">
                    <CardTitle className="flex items-center gap-2">
                      <Heart className="text-red-500" />
                      Draft Controls
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="p-4">
                    {selectedPlayer ? (
                      <div className="space-y-4">
                        <div className="text-center">
                          <img
                            src={selectedPlayer.image || '/placeholder.svg'}
                            alt={selectedPlayer.name}
                            className="w-24 h-24 mx-auto mb-3 rounded-xl object-cover border-2 border-pink-200 shadow-sm"
                          />
                          <div className="font-semibold">{selectedPlayer.name}</div>
                          <div className="text-sm text-gray-600">
                            {selectedPlayer.team} • {selectedPlayer.position}
                          </div>
                        </div>

                        <div className="text-center">
                          <div className="text-2xl font-bold text-purple-600">
                            {selectedPlayer.points}
                          </div>
                          <div className="text-sm text-gray-500">Cuddle Points</div>
                        </div>

                        <Button
                          onClick={() => draftPlayer(selectedPlayer)}
                          className="w-full bg-gradient-to-r from-pink-500 to-purple-600 hover:from-pink-600 hover:to-purple-700 text-white font-semibold rounded-full"
                        >
                          Draft Jellycat
                        </Button>
                        {profile && profile.id === selectedPlayer?.id && !profile.error && (
                          <div className="mt-4 space-y-2 text-xs bg-purple-50 p-3 rounded border border-purple-200">
                            <div className="font-semibold text-purple-700 mb-1">
                              Performance Metrics
                            </div>
                            <div className="grid grid-cols-2 gap-2">
                              <div className="bg-white rounded p-2 border">
                                <div className="text-[10px] text-gray-500">Consistency</div>
                                <div className="font-semibold text-purple-600">
                                  {profile.metrics.consistency}%
                                </div>
                              </div>
                              <div className="bg-white rounded p-2 border">
                                <div className="text-[10px] text-gray-500">Popularity</div>
                                <div className="font-semibold text-purple-600">
                                  {profile.metrics.popularity}%
                                </div>
                              </div>
                              <div className="bg-white rounded p-2 border">
                                <div className="text-[10px] text-gray-500">Efficiency</div>
                                <div className="font-semibold text-purple-600">
                                  {profile.metrics.efficiency}%
                                </div>
                              </div>
                              <div className="bg-white rounded p-2 border">
                                <div className="text-[10px] text-gray-500">Trend Δ</div>
                                <div
                                  className={`font-semibold ${Number(profile.metrics.trendDelta) >= 0 ? 'text-green-600' : 'text-red-600'}`}
                                >
                                  {profile.metrics.trendDelta}
                                </div>
                              </div>
                            </div>
                            <div className="text-[10px] text-gray-500">
                              (Demo metrics derived locally. Real build would stream from ClickHouse
                              aggregates.)
                            </div>
                          </div>
                        )}
                      </div>
                    ) : (
                      <div className="text-center text-gray-500 py-8">
                        <div className="text-4xl mb-2">🧸</div>
                        <div>Select a Jellycat to draft</div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="teams">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {teams.map((team) => {
                const stats = getTeamStats(team)
                return (
                  <Card
                    key={team.id}
                    className={`border-2 ${team.color.replace('bg-', 'border-').replace('-100', '-300')}`}
                  >
                    <CardHeader className={team.color}>
                      <CardTitle className="flex items-center gap-3">
                        <span className="text-2xl">{team.mascot}</span>
                        <div>
                          <div>{team.name}</div>
                          <div className="text-sm font-normal text-gray-600">
                            Owner: {team.owner} • {stats.totalCuddlePoints} pts
                          </div>
                        </div>
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="p-4">
                      {team.players.length > 0 ? (
                        <div className="space-y-3">
                          {team.players.map((player) => (
                            <div
                              key={player.id}
                              className="flex items-center gap-3 p-2 bg-white rounded-lg border"
                            >
                              <img
                                src={player.image || '/placeholder.svg'}
                                alt={player.name}
                                className="w-10 h-10 rounded-lg object-cover border border-pink-200"
                              />
                              <div className="flex-1">
                                <div className="font-semibold text-sm">{player.name}</div>
                                <div className="text-xs text-gray-600">
                                  {player.team} • {player.position}
                                </div>
                              </div>
                              <div className="text-sm font-semibold text-purple-600">
                                {player.points}
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="text-center text-gray-500 py-8">
                          <div className="text-3xl mb-2">💤</div>
                          <div className="text-sm">No Jellycats drafted yet</div>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          </TabsContent>

          <TabsContent value="chat">
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <div className="lg:col-span-2">
                <Card className="border-2 border-blue-200">
                  <CardHeader className="bg-blue-50">
                    <CardTitle>Live Chat & Reactions</CardTitle>
                  </CardHeader>
                  <CardContent className="p-4 space-y-3">
                    {chat.length === 0 && <div className="text-gray-500">No messages yet</div>}
                    {chat.map((m) => (
                      <div key={m.id} className="p-3 rounded-lg bg-white border">
                        <div className="text-sm">
                          <span
                            className={
                              m.type === 'system'
                                ? 'text-purple-700 font-semibold'
                                : 'text-gray-800'
                            }
                          >
                            {m.type === 'system' ? 'System' : 'User'}
                          </span>
                          <span className="text-gray-500">
                            {' '}
                            • {new Date(m.ts).toLocaleTimeString()}
                          </span>
                        </div>
                        <div>{m.text}</div>
                        <div className="mt-2 flex gap-2">
                          {['🧸', '🎉', '🔥', '👏', '😮', '😂', '❤️'].map((e) => (
                            <button
                              key={e}
                              className="px-2 py-1 border rounded bg-white hover:bg-gray-50"
                              onClick={async () => {
                                let user = 'anon'
                                try {
                                  const stored =
                                    localStorage.getItem('jellycat_owner') ||
                                    localStorage.getItem('jellycat_session')
                                  if (!stored) {
                                    const sid = 'sess_' + Math.random().toString(36).slice(2, 10)
                                    localStorage.setItem('jellycat_session', sid)
                                    user = sid
                                  } else user = stored
                                } catch {}
                                await trpc.chat.react.mutate({ messageId: m.id, emote: e, user })
                                await loadState()
                              }}
                            >
                              <span className="mr-1">{e}</span>
                              <span className="text-xs text-gray-600">{m.emotes[e] || 0}</span>
                            </button>
                          ))}
                        </div>
                      </div>
                    ))}
                  </CardContent>
                </Card>
              </div>
              <div>
                <ChatInput onSent={loadState} />
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}

function ChatInput({ onSent }: { onSent: () => void }) {
  const [text, setText] = useState('')
  const [owner, setOwner] = useState('')
  useEffect(() => {
    try {
      const o = localStorage.getItem('jellycat_owner')
      if (o) setOwner(o)
    } catch {}
  }, [])
  return (
    <Card className="border-2 border-blue-200 sticky top-4">
      <CardHeader className="bg-blue-50">
        <CardTitle>Send a message</CardTitle>
      </CardHeader>
      <CardContent className="p-4">
        <div className="flex gap-2">
          <input
            className="flex-1 border rounded px-3 py-2"
            placeholder="Cheer on picks…"
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={async (e) => {
              if (e.key === 'Enter') {
                await send()
              }
            }}
          />
          <button className="px-4 py-2 rounded bg-blue-600 text-white" onClick={send}>
            Send
          </button>
        </div>
      </CardContent>
    </Card>
  )

  async function send() {
    if (!text.trim()) return
    const prefix = owner ? `${owner}: ` : ''
    await trpc.chat.send.mutate({ text: prefix + text })
    setText('')
    onSent()
  }
}
