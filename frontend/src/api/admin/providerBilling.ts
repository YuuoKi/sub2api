import { apiClient } from '../client'

export type ProviderBillingConclusion = 'reconciled' | 'has_diff' | 'not_uploaded'

export interface ProviderBillingImportHeader {
  provider: string
  provider_account_id: string
  billing_period_start: string
  billing_period_end: string
  timezone: string
  original_currency: 'CNY' | 'USD'
  source_type: 'csv' | 'xlsx'
  invoice_number?: string
}

export interface ProviderBillingImport {
  id: number
  provider: string
  provider_account_id: string
  billing_period_start: string
  billing_period_end: string
  timezone: string
  original_currency: string
  source_type: string
  invoice_number?: string
  file_sha256: string
  storage_key: string
  original_filename: string
  byte_size: number
  status: string
  line_count: number
  created_at: string
}

export interface ProviderBillingMatch {
  id: number
  import_id: number
  external_line_id: string
  match_status: string
  match_mode: string
  internal_ref_type?: string
  internal_ref_id?: string
  provider_amount: string
  internal_amount: string
  provider_usage: string
  internal_usage: string
  currency: string
  model: string
  sku: string
  account_day?: string
  diff?: Record<string, unknown>
}

export interface ProviderBillingPeriodSummary {
  provider: string
  provider_account_id: string
  billing_period_start: string
  billing_period_end: string
  import_count: number
  matched: number
  has_diff: number
  provider_only: number
  internal_only: number
  conclusion: ProviderBillingConclusion
}

export interface ProviderBillingBossConclusion {
  provider?: string
  provider_account_id?: string
  billing_period_start?: string
  billing_period_end?: string
  conclusion: ProviderBillingConclusion
}

function buildFormData(header: ProviderBillingImportHeader, file: File): FormData {
  const form = new FormData()
  form.append('provider', header.provider)
  form.append('provider_account_id', header.provider_account_id)
  form.append('billing_period_start', header.billing_period_start)
  form.append('billing_period_end', header.billing_period_end)
  form.append('timezone', header.timezone)
  form.append('original_currency', header.original_currency)
  form.append('source_type', header.source_type)
  if (header.invoice_number) {
    form.append('invoice_number', header.invoice_number)
  }
  form.append('file', file)
  return form
}

export async function listImports(provider?: string): Promise<{ items: ProviderBillingImport[] }> {
  const { data } = await apiClient.get<{ items: ProviderBillingImport[] }>('/admin/provider-billing/imports', {
    params: provider ? { provider } : undefined
  })
  return data
}

export async function getImport(id: number): Promise<{ import: ProviderBillingImport; lines: unknown[] }> {
  const { data } = await apiClient.get<{ import: ProviderBillingImport; lines: unknown[] }>(
    `/admin/provider-billing/imports/${id}`
  )
  return data
}

export async function previewImport(
  header: ProviderBillingImportHeader,
  file: File
): Promise<{ file_sha256: string; line_count: number; lines: unknown[]; duplicate: boolean }> {
  const { data } = await apiClient.post('/admin/provider-billing/imports/preview', buildFormData(header, file))
  return data
}

export async function importStatement(
  header: ProviderBillingImportHeader,
  file: File
): Promise<{ import: ProviderBillingImport; matches: ProviderBillingMatch[] }> {
  const { data } = await apiClient.post('/admin/provider-billing/imports', buildFormData(header, file))
  return data
}

export async function listMatches(
  importId: number,
  status?: string
): Promise<{ items: ProviderBillingMatch[] }> {
  const { data } = await apiClient.get<{ items: ProviderBillingMatch[] }>(
    `/admin/provider-billing/imports/${importId}/matches`,
    { params: status ? { status } : undefined }
  )
  return data
}

export async function getPeriodSummary(params?: {
  start?: string
  end?: string
}): Promise<{ items: ProviderBillingPeriodSummary[] }> {
  const { data } = await apiClient.get<{ items: ProviderBillingPeriodSummary[] }>(
    '/admin/provider-billing/period-summary',
    { params }
  )
  return data
}

export async function getBossConclusions(): Promise<{ items: ProviderBillingBossConclusion[] }> {
  const { data } = await apiClient.get<{ items: ProviderBillingBossConclusion[] }>(
    '/admin/provider-billing/boss-conclusions'
  )
  return data
}

export function matchesExportUrl(importId: number): string {
  return `/api/v1/admin/provider-billing/imports/${importId}/matches.csv`
}

const providerBillingAPI = {
  listImports,
  getImport,
  previewImport,
  importStatement,
  listMatches,
  getPeriodSummary,
  getBossConclusions,
  matchesExportUrl
}

export default providerBillingAPI
