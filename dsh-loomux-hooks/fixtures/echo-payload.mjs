// Fixture hook: read the JSON payload from stdin, echo it back on stdout.
// It lives outside test/ because the test runner would otherwise run every
// .mjs file under test/ as a test file, and this one blocks on stdin.
import { readFileSync } from 'node:fs'

const payload = JSON.parse(readFileSync(0, 'utf8'))
process.stdout.write(JSON.stringify({ received: payload }))
