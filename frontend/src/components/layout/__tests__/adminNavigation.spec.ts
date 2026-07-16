import { describe, expect, it } from 'vitest'
import { filterAdminNavigationForMode } from '../adminNavigation'

describe('admin video navigation policy', () => {
  it('keeps video administration visible in simple mode filtering', () => {
    const items = [
      { path: '/admin/users', hideInSimpleMode: true },
      { path: '/admin/video', hideInSimpleMode: false },
    ]
    expect(filterAdminNavigationForMode(items, true).map(item => item.path)).toEqual(['/admin/video'])
  })
})
