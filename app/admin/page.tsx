'use client'

import { useEffect, useMemo, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { Player, Team, Tier } from '@/lib/draft-store'
import { trpc } from '@/lib/trpc-client'

type DraftState = { players: Player[]; teams: Team[] }

export default function AdminPage() {
  // Note: middleware enforces access, this is defense-in-depth if client renders without groups
  const [unauthorized, setUnauthorized] = useState(false)
  const [state, setState] = useState<DraftState>({ players: [], teams: [] })
  const [loading, setLoading] = useState(true)

  async function load() {
    setLoading(true)
    const data = await trpc.draft.state.query()
    setState(data)
    setLoading(false)
  }

  useEffect(() => {
    load().catch(() => setUnauthorized(true))
  }, [])

  // Add Jellycat form
  const [form, setForm] = useState({
    name: '',
    position: 'CC',
    team: '',
    points: 200,
    tier: 'A' as Tier,
    image: '',
  })
  const addJellycat = async () => {
    await trpc.players.add.mutate({ ...form, points: Number(form.points) })
    setForm({ name: '', position: 'CC', team: '', points: 200, tier: 'A', image: '' })
    await load()
  }

  // Update points
  const [selectedPlayerId, setSelectedPlayerId] = useState<string>('')
  const [points, setPoints] = useState<number>(200)
  const updatePoints = async () => {
    if (!selectedPlayerId) return
    await trpc.players.setPoints.mutate({ id: selectedPlayerId, points })
    await load()
  }

  // Reorder teams (simple up/down controls)
  const moveTeam = async (index: number, dir: -1 | 1) => {
    const next = [...state.teams]
    const j = index + dir
    if (j < 0 || j >= next.length) return
    ;[next[index], next[j]] = [next[j], next[index]]
    setState({ ...state, teams: next })
    const order = next.map((t) => t.id)
    await trpc.teams.reorder.mutate({ order })
    await load()
  }

  const availablePlayers = useMemo(() => state.players.filter((p) => !p.drafted), [state.players])

  if (loading) return <div className="p-6">Loading…</div>
  if (unauthorized) return <div className="p-6">Not authorized</div>

  return (
    <div className="min-h-screen bg-gradient-to-br from-pink-50 via-purple-50 to-blue-50 p-6">
      <div className="container mx-auto">
        <h1 className="text-3xl font-bold mb-6">Admin Dashboard</h1>
        <Tabs defaultValue="jellycats" className="w-full">
          <TabsList className="mb-4 bg-white border">
            <TabsTrigger value="jellycats">Add Jellycats</TabsTrigger>
            <TabsTrigger value="teams">Reorder Teams</TabsTrigger>
            <TabsTrigger value="points">Update Cuddle Points</TabsTrigger>
          </TabsList>

          <TabsContent value="jellycats">
            <Card className="border-2 border-pink-200">
              <CardHeader className="bg-pink-50">
                <CardTitle>Add a Jellycat</CardTitle>
              </CardHeader>
              <CardContent className="p-4 space-y-3">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div>
                    <Label htmlFor="name">Name</Label>
                    <Input
                      id="name"
                      value={form.name}
                      onChange={(e) => setForm({ ...form, name: e.target.value })}
                    />
                  </div>
                  <div>
                    <Label htmlFor="team">Team</Label>
                    <Input
                      id="team"
                      value={form.team}
                      onChange={(e) => setForm({ ...form, team: e.target.value })}
                    />
                  </div>
                  <div>
                    <Label htmlFor="position">Position (CC/SS/HH/CH)</Label>
                    <Input
                      id="position"
                      value={form.position}
                      onChange={(e) => setForm({ ...form, position: e.target.value })}
                    />
                  </div>
                  <div>
                    <Label htmlFor="points">Cuddle Points</Label>
                    <Input
                      id="points"
                      type="number"
                      value={form.points}
                      onChange={(e) => setForm({ ...form, points: Number(e.target.value) })}
                    />
                  </div>
                  <div>
                    <Label htmlFor="tier">Tier (S/A/B/C)</Label>
                    <Input
                      id="tier"
                      value={form.tier}
                      onChange={(e) => setForm({ ...form, tier: e.target.value as Tier })}
                    />
                  </div>
                  <div>
                    <Label htmlFor="image">Image URL</Label>
                    <Input
                      id="image"
                      value={form.image}
                      onChange={(e) => setForm({ ...form, image: e.target.value })}
                      placeholder="/jellycats/...png"
                    />
                  </div>
                </div>
                <Button onClick={addJellycat} className="mt-2">
                  Add Jellycat
                </Button>
              </CardContent>
            </Card>

            <Card className="mt-4 border-2 border-purple-200">
              <CardHeader className="bg-purple-50">
                <CardTitle>Available Jellycats ({availablePlayers.length})</CardTitle>
              </CardHeader>
              <CardContent className="p-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                {availablePlayers.map((p) => (
                  <div key={p.id} className="p-3 rounded-lg bg-white border">
                    <div className="flex gap-3 items-center">
                      <img
                        src={p.image || '/placeholder.svg'}
                        alt={p.name}
                        className="w-12 h-12 rounded-md border"
                      />
                      <div className="flex-1">
                        <div className="font-semibold text-sm">{p.name}</div>
                        <div className="text-xs text-gray-600">
                          {p.team} • {p.position}
                        </div>
                      </div>
                      <Badge>{p.tier}</Badge>
                      <div className="text-purple-600 font-semibold">{p.points}</div>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="teams">
            <Card className="border-2 border-amber-200">
              <CardHeader className="bg-amber-50">
                <CardTitle>Reorder Teams</CardTitle>
              </CardHeader>
              <CardContent className="p-4 space-y-3">
                {state.teams.map((t, i) => (
                  <div
                    key={t.id}
                    className={`flex items-center justify-between p-3 rounded-lg border ${t.color}`}
                  >
                    <div className="flex items-center gap-3">
                      <span className="text-xl">{t.mascot}</span>
                      <div>
                        <div className="font-semibold">{t.name}</div>
                        <div className="text-xs text-gray-600">Owner: {t.owner}</div>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button variant="outline" onClick={() => moveTeam(i, -1)} disabled={i === 0}>
                        ↑
                      </Button>
                      <Button
                        variant="outline"
                        onClick={() => moveTeam(i, 1)}
                        disabled={i === state.teams.length - 1}
                      >
                        ↓
                      </Button>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="points">
            <Card className="border-2 border-blue-200">
              <CardHeader className="bg-blue-50">
                <CardTitle>Update Cuddle Points</CardTitle>
              </CardHeader>
              <CardContent className="p-4 space-y-3">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3 items-end">
                  <div>
                    <Label htmlFor="player">Player ID</Label>
                    <Input
                      id="player"
                      value={selectedPlayerId}
                      onChange={(e) => setSelectedPlayerId(e.target.value)}
                      placeholder="e.g. 1 or player_xxx"
                    />
                  </div>
                  <div>
                    <Label htmlFor="pts">Points</Label>
                    <Input
                      id="pts"
                      type="number"
                      value={points}
                      onChange={(e) => setPoints(Number(e.target.value))}
                    />
                  </div>
                  <Button onClick={updatePoints}>Save</Button>
                </div>

                <div className="mt-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                  {state.players.map((p) => (
                    <div key={p.id} className="p-3 rounded-lg bg-white border">
                      <div className="flex gap-3 items-center">
                        <img
                          src={p.image || '/placeholder.svg'}
                          alt={p.name}
                          className="w-10 h-10 rounded-md border"
                        />
                        <div className="flex-1">
                          <div className="font-semibold text-sm">{p.name}</div>
                          <div className="text-xs text-gray-600">ID: {p.id}</div>
                        </div>
                        <div className="text-purple-600 font-semibold">{p.points}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
