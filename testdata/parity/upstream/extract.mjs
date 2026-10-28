// Extracts MJML documents embedded in the upstream MJML test suite.
//
// Usage:
//   git clone --depth 1 --filter=blob:none --sparse https://github.com/mjmlio/mjml /tmp/mjml-upstream
//   git -C /tmp/mjml-upstream sparse-checkout set packages
//   node testdata/parity/upstream/extract.mjs /tmp/mjml-upstream > /tmp/fixtures-list.txt
//
// It prints the extracted files (name, source test, text) as JSON lines; the
// curated subset committed here was then filtered with the reference CLI:
// documents that MJML 4.18 only renders by ignoring illegal attributes or
// unknown tags are excluded, because this implementation validates them.

import fs from 'node:fs'
import path from 'node:path'

const upstream = process.argv[2]
if (!upstream) {
  console.error('usage: extract.mjs <path-to-upstream-mjml-checkout>')
  process.exit(1)
}

const packagesDir = path.join(upstream, 'packages')
const seen = new Set()

for (const pkg of fs.readdirSync(packagesDir, { withFileTypes: true })) {
  if (!pkg.isDirectory()) continue

  const testDir = path.join(packagesDir, pkg.name, 'test')
  if (!fs.existsSync(testDir)) continue

  for (const testFile of fs.readdirSync(testDir).filter((f) => f.endsWith('.test.js'))) {
    const source = fs.readFileSync(path.join(testDir, testFile), 'utf8')
    const matches = source.match(/<mjml[\s\S]*?<\/mjml>/g) ?? []

    matches.forEach((match, index) => {
      if (match.includes('${') || match.includes('\\`')) return
      if (seen.has(match.trim())) return
      seen.add(match.trim())

      console.log(
        JSON.stringify({
          name: `${pkg.name.replace('mjml-', '')}--${testFile.replace('.test.js', '')}--${String(index).padStart(2, '0')}.mjml`,
          source: `packages/${pkg.name}/test/${testFile}`,
          mjml: match,
        }),
      )
    })
  }
}
