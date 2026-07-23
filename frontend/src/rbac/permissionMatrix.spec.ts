import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { pagePermissionMatrix } from './permissionMatrix'

describe('pagePermissionMatrix', () => {
  it('has one classification per path', () => {
    const paths = pagePermissionMatrix.map((item) => item.path)
    expect(new Set(paths).size).toBe(paths.length)
  })

  it('assigns a permission to every rbac page', () => {
    expect(
      pagePermissionMatrix
        .filter((item) => item.classification === 'rbac')
        .every((item) => Boolean(item.permission)),
    ).toBe(true)
  })

  it('keeps public pages permission-free', () => {
    expect(
      pagePermissionMatrix
        .filter((item) => item.classification === 'public')
        .every((item) => item.permission === null),
    ).toBe(true)
  })

  it('classifies every statically declared Vue route', () => {
    const routerSource = readFileSync(
      resolve(process.cwd(), 'src/router/index.ts'),
      'utf8',
    )
    const declaredPaths = [
      ...routerSource.matchAll(/^\s{4}path:\s*['"]([^'"]+)['"],?$/gm),
    ].map((match) => match[1])
    const classifiedPaths = new Set(pagePermissionMatrix.map((item) => item.path))

    expect(declaredPaths.length).toBeGreaterThan(0)
    expect(declaredPaths.filter((path) => !classifiedPaths.has(path))).toEqual([])
  })
})
