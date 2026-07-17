import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import AnimatedEmptyState from '../AnimatedEmptyState.vue'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AnimatedEmptyState.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AnimatedEmptyState behavior', () => {
  it('renders title and description', () => {
    const wrapper = mount(AnimatedEmptyState, {
      props: { variant: 'video-tasks', title: '当前还没有任务记录。', description: '可以先试跑一条任务。' },
    })
    expect(wrapper.text()).toContain('当前还没有任务记录。')
    expect(wrapper.text()).toContain('可以先试跑一条任务。')
    expect(wrapper.attributes('data-variant')).toBe('video-tasks')
  })

  it('emits action when the action button is clicked', async () => {
    const wrapper = mount(AnimatedEmptyState, {
      props: { title: '空', actionLabel: '试跑一条任务' },
    })
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    await button.trigger('click')
    expect(wrapper.emitted('action')).toHaveLength(1)
  })

  it('hides the action button without actionLabel', () => {
    const wrapper = mount(AnimatedEmptyState, {
      props: { title: '空' },
    })
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('defaults to the generic variant', () => {
    const wrapper = mount(AnimatedEmptyState, {
      props: { title: '空' },
    })
    expect(wrapper.attributes('data-variant')).toBe('generic')
  })

  it('surfaces the canonical product brand on empty states', () => {
    const wrapper = mount(AnimatedEmptyState, {
      props: { title: '空' },
    })
    expect(wrapper.get('[data-testid="empty-state-brand"]').text()).toBe('无界 · 企业 AI 中台')
  })

  it('does not expose the unused video-dashboard variant', () => {
    expect(componentSource).not.toContain("variant === 'video-dashboard'")
    expect(componentSource).not.toContain("'video-dashboard'")
  })
})

describe('AnimatedEmptyState motion source contract', () => {
  it('uses slow ambient animation classes on the illustration', () => {
    expect(componentSource).toContain('ui-anim-float')
    expect(componentSource).toContain('ui-anim-breathe')
  })

  it('keeps teal accents restrained', () => {
    expect(componentSource).toContain('text-teal-500 dark:text-teal-300')
  })

  it('freezes ambient animations under prefers-reduced-motion', () => {
    const reducedMotionBlock = styleSource.match(/@media \(prefers-reduced-motion: reduce\) \{[\s\S]*?\.ui-anim-float,[\s\S]*?\n {2}\}/)
    expect(reducedMotionBlock).not.toBeNull()
    expect(reducedMotionBlock?.[0]).toContain('.ui-anim-breathe')
    expect(reducedMotionBlock?.[0]).toContain('animation: none;')
  })

  it('defines 3-4s ambient keyframes in style.css', () => {
    expect(styleSource).toContain('@keyframes ui-float')
    expect(styleSource).toContain('@keyframes ui-breathe')
    expect(styleSource).toMatch(/\.ui-anim-float \{[\s\S]*?3\.\ds ease-in-out infinite;/)
    expect(styleSource).toMatch(/\.ui-anim-breathe \{[\s\S]*?3\.\ds ease-in-out infinite;/)
  })
})
