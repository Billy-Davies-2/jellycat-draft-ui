// Server-only helper for SQLite demo seeding of jellycat image players
export function seedJellycats(db: any, insPlayer: any) {
  try {
    const path = require('path')
    const fs = require('fs')
    const jellyDir = path.join(process.cwd(), 'public', 'jellycats')
    if (!fs.existsSync(jellyDir)) return
    const files: string[] = fs
      .readdirSync(jellyDir)
      .filter((f: string) => /\.(png|jpe?g|svg)$/i.test(f))
    for (const file of files) {
      const base = file.replace(/\.[^.]+$/, '')
      const id = `jcat_${base}`
      const exists = db.query('SELECT id FROM players WHERE id=?').get(id)
      if (exists) continue
      const display = base.replace(/[-_]+/g, ' ').replace(/\b\w/g, (c: string) => c.toUpperCase())
      insPlayer.run(id, display, 'PLUSH', 'Jellycat', 0, 'Z', 0, `/jellycats/${file}`)
    }
  } catch {
    // ignore
  }
}
