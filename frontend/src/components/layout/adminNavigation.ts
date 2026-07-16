export interface SimpleModeNavigationItem {
  hideInSimpleMode?: boolean
}

export const filterAdminNavigationForMode = <T extends SimpleModeNavigationItem>(
  items: T[],
  isSimpleMode: boolean,
): T[] => (isSimpleMode ? items.filter(item => !item.hideInSimpleMode) : items)
