//客户状态

export const CustomerStatus:string[] = [
  '潜在',
  '活动',
  '停用',
  '冻结',
  '黑名单',
  '合同到期',
]

export const CustomerTypes:string[]   = [
  '个人',
  '企业',
  '组织',
]

// 客户交易类型：提交给后端的 value 使用英文 code，中文仅用于展示。
export const CustomerTransactionTypes = [
  { label: '回款', value: 'payment', direction: 'receivable_decrease' },
  { label: '退货冲减', value: 'return_credit', direction: 'receivable_decrease' },
]

export const CustomerTransactionTypeLabels: Record<string, string> = CustomerTransactionTypes.reduce((map, one) => {
  map[one.value] = one.label
  return map
}, {} as Record<string, string>)
