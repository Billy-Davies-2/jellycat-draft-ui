'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useRouter } from 'next/navigation'
import { trpc } from '@/lib/trpc-client'

export default function StartPage() {
  const [name, setName] = useState('')
  const [owner, setOwner] = useState('')

  // Load persisted owner (simple localStorage mock of auth)
  useEffect(() => {
    try {
      const o = localStorage.getItem('jellycat_owner')
      if (o) setOwner(o)
    } catch {}
  }, [])
  const [joining, setJoining] = useState(false)
  const router = useRouter()

  const create = async () => {
    if (!name.trim()) return
    setJoining(true)
    try {
      await trpc.teams.add.mutate({ name, owner })
      try {
        if (owner) localStorage.setItem('jellycat_owner', owner)
      } catch {}
      // Small delay to allow WS/event + state persistence before navigation
      await trpc.draft.state.query()
      setTimeout(() => router.push('/draft'), 50)
    } catch (e) {
      setJoining(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-pink-50 via-purple-50 to-blue-50">
      <div className="container mx-auto px-4 py-12">
        <Card className="max-w-xl mx-auto border-2 border-pink-200">
          <CardHeader className="bg-gradient-to-r from-pink-100 to-purple-100">
            <CardTitle className="text-center">Create Your Team</CardTitle>
          </CardHeader>
          <CardContent className="p-6 space-y-4">
            <div>
              <Label htmlFor="name">Team Name</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Snuggly Sloths"
              />
            </div>
            <div>
              <Label htmlFor="owner">Owner (optional)</Label>
              <Input
                id="owner"
                value={owner}
                onChange={(e) => setOwner(e.target.value)}
                placeholder="Your name"
              />
            </div>
            <Button className="w-full" onClick={create} disabled={!name.trim()}>
              {joining ? 'Joining…' : 'Create Team and Join Draft'}
            </Button>
            <Button className="w-full" variant="outline" onClick={() => router.push('/draft')}>
              Go to Draft
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
