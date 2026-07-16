import type { DriveStep } from 'driver.js'

type Translate = (key: string) => string

const operatorStep = (
  element: string,
  titleKey: string,
  descriptionKey: string,
  side: 'right' | 'bottom' = 'right'
): DriveStep => ({
  element,
  popover: {
    title: titleKey,
    description: descriptionKey,
    side,
    align: 'center',
    showButtons: ['next', 'previous', 'close']
  }
})

/**
 * 无界内部管理首次引导。
 * 只解释老板真实要完成的四个业务环节，不在引导中代替用户创建或提交数据。
 */
export const getAdminSteps = (t: Translate, _isSimpleMode = false): DriveStep[] => [
  operatorStep(
    '#sidebar-user-manage',
    t('onboarding.admin.employee.title'),
    t('onboarding.admin.employee.description')
  ),
  operatorStep(
    '#sidebar-group-manage',
    t('onboarding.admin.group.title'),
    t('onboarding.admin.group.description')
  ),
  operatorStep(
    '#sidebar-channel-manage',
    t('onboarding.admin.account.title'),
    t('onboarding.admin.account.description')
  ),
  operatorStep(
    '#sidebar-global-usage',
    t('onboarding.admin.usage.title'),
    t('onboarding.admin.usage.description')
  )
]

/** 员工只需要知道在哪里核对本人真实用量。 */
export const getUserSteps = (t: Translate): DriveStep[] => [
  {
    element: '#sidebar-my-usage',
    popover: {
      title: t('onboarding.user.usage.title'),
      description: t('onboarding.user.usage.description'),
      side: 'right',
      align: 'center',
      showButtons: ['close']
    }
  }
]
