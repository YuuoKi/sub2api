import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import TaskProgressRing from '../TaskProgressRing.vue'

const CIRCUMFERENCE = 2 * Math.PI * 16

describe('TaskProgressRing indeterminate mode (default, no fake percentage)', () => {
  it('renders a rotating gap arc with the phase text in the center', () => {
    const wrapper = mount(TaskProgressRing, {
      props: { phase: '生成中' },
    })
    const arc = wrapper.findAll('circle')[1]
    expect(arc.attributes('stroke-dasharray')).toBe(`${CIRCUMFERENCE * 0.28} ${CIRCUMFERENCE * 0.72}`)
    expect(arc.classes()).toContain('ui-anim-ring')
    expect(wrapper.text()).toContain('生成中')
    expect(wrapper.text()).not.toContain('%')
  })

  it('exposes an accessible status label without a percentage', () => {
    const wrapper = mount(TaskProgressRing, {
      props: { phase: '排队中' },
    })
    expect(wrapper.attributes('role')).toBe('status')
    expect(wrapper.attributes('aria-label')).toBe('排队中')
  })

  it('moves the phase beside the ring when sideLabel is set', () => {
    const wrapper = mount(TaskProgressRing, {
      props: { phase: '即将完成', sideLabel: true },
    })
    expect(wrapper.find('text').exists()).toBe(false)
    expect(wrapper.text()).toContain('即将完成')
  })
})

describe('TaskProgressRing determinate mode (real progress only)', () => {
  it('shows a determinate arc and honest percentage when progress is provided', () => {
    const wrapper = mount(TaskProgressRing, {
      props: { phase: '下载中', progress: 0.5 },
    })
    const arc = wrapper.findAll('circle')[1]
    expect(arc.attributes('stroke-dasharray')).toBe(`${CIRCUMFERENCE}`)
    expect(Number(arc.attributes('stroke-dashoffset'))).toBeCloseTo(CIRCUMFERENCE * 0.5, 5)
    expect(arc.classes()).not.toContain('ui-anim-ring')
    expect(wrapper.text()).toContain('50%')
    expect(wrapper.attributes('aria-label')).toBe('下载中 50%')
  })

  it('clamps out-of-range progress instead of fabricating values', () => {
    const wrapper = mount(TaskProgressRing, {
      props: { progress: 1.4 },
    })
    expect(wrapper.text()).toContain('100%')
  })
})
