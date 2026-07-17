import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Skeleton from '../Skeleton.vue'

const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')
const tailwindConfigPath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../../tailwind.config.js')
const tailwindConfigSource = readFileSync(tailwindConfigPath, 'utf8')

describe('Skeleton shimmer', () => {
  it('renders with the shimmer utility class instead of a bare pulse', () => {
    const wrapper = mount(Skeleton, { props: { width: 120, height: 16 } })
    expect(wrapper.classes()).toContain('ui-skeleton')
    expect(wrapper.classes()).not.toContain('animate-pulse')
  })

  it('enables the previously idle shimmer keyframes from tailwind.config.js', () => {
    expect(tailwindConfigSource).toContain('shimmer')
    const skeletonBlock = styleSource.match(/\.ui-skeleton \{[\s\S]*?\n {2}\}/)
    expect(skeletonBlock).not.toBeNull()
    expect(skeletonBlock?.[0]).toContain('animate-shimmer')
    expect(skeletonBlock?.[0]).toContain('linear-gradient')
  })

  it('falls back to a static gray block under prefers-reduced-motion', () => {
    const reducedMotionBlock = styleSource.match(/@media \(prefers-reduced-motion: reduce\) \{[\s\S]*?\.ui-skeleton \{[\s\S]*?\n {4}\}/)
    expect(reducedMotionBlock).not.toBeNull()
    expect(reducedMotionBlock?.[0]).toContain('animation: none;')
  })
})
