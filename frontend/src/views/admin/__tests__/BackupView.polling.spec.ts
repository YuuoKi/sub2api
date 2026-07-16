import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import BackupView from '../BackupView.vue'

const {
  backupAPI,
  showSuccess,
  showError,
  showWarning,
  requestConfirmation,
  requestTextPrompt,
} = vi.hoisted(() => ({
  backupAPI: {
    getS3Config: vi.fn(),
    updateS3Config: vi.fn(),
    testS3Connection: vi.fn(),
    getSchedule: vi.fn(),
    updateSchedule: vi.fn(),
    createBackup: vi.fn(),
    listBackups: vi.fn(),
    getBackup: vi.fn(),
    deleteBackup: vi.fn(),
    getDownloadURL: vi.fn(),
    restoreBackup: vi.fn(),
  },
  showSuccess: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
  requestConfirmation: vi.fn(),
  requestTextPrompt: vi.fn(),
}))

vi.mock('@/api', () => ({ adminAPI: { backup: backupAPI } }))
vi.mock('@/stores', () => ({
  useAppStore: () => ({ showSuccess, showError, showWarning }),
}))
vi.mock('@/composables/useAppDialog', () => ({ requestConfirmation, requestTextPrompt }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const completedRecord = {
  id: 'backup-1',
  status: 'completed' as const,
  backup_type: 'manual',
  file_name: 'backup.sql',
  s3_key: 'backups/backup.sql',
  size_bytes: 10,
  triggered_by: 'manual',
  started_at: '2026-07-16T00:00:00Z',
}

const runningRecord = { ...completedRecord, status: 'running' as const }

async function mountView(records = [] as Array<typeof completedRecord>) {
  backupAPI.getS3Config.mockResolvedValue({
    endpoint: '', region: 'auto', bucket: '', access_key_id: '', prefix: 'backups/', force_path_style: false,
  })
  backupAPI.getSchedule.mockResolvedValue({ enabled: false, cron_expr: '0 2 * * *', retain_days: 14, retain_count: 10 })
  backupAPI.listBackups.mockResolvedValue({ items: records })
  const wrapper = mount(BackupView)
  await flushPromises()
  return wrapper
}

describe('BackupView operation polling', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    Object.values(backupAPI).forEach(mock => mock.mockReset())
    showSuccess.mockReset()
    showError.mockReset()
    showWarning.mockReset()
    requestConfirmation.mockReset()
    requestTextPrompt.mockReset()
    requestConfirmation.mockResolvedValue(true)
    requestTextPrompt.mockResolvedValue('admin-password')
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('stops backup polling after three consecutive failures and offers retry with the real reason', async () => {
    backupAPI.createBackup.mockResolvedValue(runningRecord)
    backupAPI.getBackup.mockRejectedValue(new Error('backup status endpoint unavailable'))
    const wrapper = await mountView()

    const createButton = wrapper.findAll('button').find(button => button.text().includes('admin.backup.operations.createBackup'))
    await createButton!.trigger('click')
    await flushPromises()
    await vi.advanceTimersByTimeAsync(6000)
    await flushPromises()

    expect(backupAPI.getBackup).toHaveBeenCalledTimes(3)
    expect(showError).toHaveBeenCalledWith('backup status endpoint unavailable')
    expect(wrapper.text()).toContain('common.unknown')
    expect(wrapper.text()).toContain('version.retry')
    wrapper.unmount()
  })

  it('stops restore polling after three consecutive failures and offers retry with the real reason', async () => {
    backupAPI.restoreBackup.mockResolvedValue({ ...completedRecord, restore_status: 'running' })
    backupAPI.getBackup.mockRejectedValue(new Error('restore status endpoint unavailable'))
    const wrapper = await mountView([completedRecord])

    const restoreButton = wrapper.findAll('button').find(button => button.text().includes('admin.backup.actions.restore'))
    await restoreButton!.trigger('click')
    await flushPromises()
    await vi.advanceTimersByTimeAsync(6000)
    await flushPromises()

    expect(backupAPI.getBackup).toHaveBeenCalledTimes(3)
    expect(showError).toHaveBeenCalledWith('restore status endpoint unavailable')
    expect(wrapper.text()).toContain('common.unknown')
    expect(wrapper.text()).toContain('version.retry')
    wrapper.unmount()
  })

  it('preserves a terminal restore failure without a success toast', async () => {
    backupAPI.restoreBackup.mockResolvedValue({ ...completedRecord, restore_status: 'running' })
    backupAPI.getBackup.mockResolvedValue({
      ...completedRecord,
      restore_status: 'failed',
      restore_error: 'database restore rejected',
    })
    const wrapper = await mountView([completedRecord])

    const restoreButton = wrapper.findAll('button').find(button => button.text().includes('admin.backup.actions.restore'))
    await restoreButton!.trigger('click')
    await flushPromises()
    await vi.advanceTimersByTimeAsync(2000)
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('database restore rejected')
    expect(showSuccess).not.toHaveBeenCalledWith('admin.backup.actions.restoreSuccess')
    wrapper.unmount()
  })

  it('clears backup polling timers when unmounted', async () => {
    backupAPI.createBackup.mockResolvedValue(runningRecord)
    backupAPI.getBackup.mockResolvedValue(runningRecord)
    const wrapper = await mountView()

    const createButton = wrapper.findAll('button').find(button => button.text().includes('admin.backup.operations.createBackup'))
    await createButton!.trigger('click')
    await flushPromises()
    expect(vi.getTimerCount()).toBeGreaterThan(0)

    wrapper.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })

  it('does not call restore API when the in-app password dialog is cancelled', async () => {
    requestTextPrompt.mockResolvedValueOnce(null)
    const wrapper = await mountView([completedRecord])

    const restoreButton = wrapper.findAll('button').find(button => button.text().includes('admin.backup.actions.restore'))
    await restoreButton!.trigger('click')
    await flushPromises()

    expect(requestTextPrompt).toHaveBeenCalledTimes(1)
    expect(backupAPI.restoreBackup).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
