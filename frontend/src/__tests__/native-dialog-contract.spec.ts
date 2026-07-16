import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

const sourceRoot = join(process.cwd(), 'src')

const sourceExtensions = new Set(['.ts', '.vue'])
const nativeDialogCall = /(?:\bwindow\s*\.\s*|(?<![\w.]))(?:confirm|prompt)\s*\(/g

const walkSourceFiles = (directory: string): string[] =>
  readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) {
      return entry.name === '__tests__' ? [] : walkSourceFiles(path)
    }

    const extension = entry.name.slice(entry.name.lastIndexOf('.'))
    return sourceExtensions.has(extension) ? [path] : []
  })

describe('native browser dialog contract', () => {
  it('does not use browser confirm or prompt in production source files', () => {
    const violations = walkSourceFiles(sourceRoot)
      .flatMap((path) => {
        const source = readFileSync(path, 'utf8')
        const executableSource = source
          .replace(/<!--[\s\S]*?-->/g, (comment) => comment.replace(/[^\n]/g, ' '))
          .replace(/\/\*[\s\S]*?\*\//g, (comment) => comment.replace(/[^\n]/g, ' '))
          .replace(/\/\/[^\n]*/g, '')
        return [...executableSource.matchAll(nativeDialogCall)].map((match) => {
          const line = executableSource.slice(0, match.index).split('\n').length
          return `${relative(sourceRoot, path)}:${line}`
        })
      })

    expect(violations).toEqual([])
  })
})
