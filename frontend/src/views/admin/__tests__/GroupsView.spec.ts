import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";

import type { AdminGroup } from "@/types";
import GroupsView from "../GroupsView.vue";

const {
  listGroups,
  getUsageSummary,
  getCapacitySummary,
  getDashboardStats,
} = vi.hoisted(() => ({
  listGroups: vi.fn(),
  getUsageSummary: vi.fn(),
  getCapacitySummary: vi.fn(),
  getDashboardStats: vi.fn(),
}));

vi.mock("@/api/admin", () => ({
  adminAPI: {
    groups: {
      list: listGroups,
      getUsageSummary,
      getCapacitySummary,
      getAll: vi.fn().mockResolvedValue([]),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      updateSortOrder: vi.fn(),
    },
    accounts: {
      list: vi.fn().mockResolvedValue({ items: [] }),
      getById: vi.fn(),
    },
    dashboard: {
      getStats: getDashboardStats,
    },
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}));

vi.mock("@/stores/onboarding", () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn(),
  }),
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  };
});

const DataTableStub = {
  props: ["columns", "data"],
  template: `
    <div>
      <div v-for="row in data" :key="row.id" data-test="group-row">
        <slot name="cell-name" :row="row" />
        <slot name="cell-billing_type" :row="row" />
        <slot name="cell-usage" :row="row" />
      </div>
    </div>
  `,
};

const createGroup = (): AdminGroup => ({
  id: 7,
  name: "Boss Group",
  description: null,
  platform: "anthropic",
  rate_multiplier: 1,
  is_exclusive: false,
  status: "active",
  subscription_type: "subscription",
  daily_limit_usd: 10,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  allow_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: "2026-07-07T00:00:00Z",
  updated_at: "2026-07-07T00:00:00Z",
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: true,
  account_count: 1,
  active_account_count: 1,
  rate_limited_account_count: 0,
  sort_order: 1,
});

describe("admin GroupsView", () => {
  beforeEach(() => {
    localStorage.clear();
    listGroups.mockReset();
    getUsageSummary.mockReset();
    getCapacitySummary.mockReset();
    getDashboardStats.mockReset();

    listGroups.mockResolvedValue({
      items: [createGroup()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    });
    getUsageSummary.mockResolvedValue([
      { group_id: 7, today_cost: 1.25, total_cost: 2.5 },
    ]);
    getCapacitySummary.mockResolvedValue([]);
    getDashboardStats.mockResolvedValue({ usd_cny_rate: 7.5 });
  });

  it("formats group usage as CNY while keeping subscription limits in USD", async () => {
    const wrapper = mount(GroupsView, {
      global: {
        stubs: {
          AppLayout: { template: "<div><slot /></div>" },
          TablePageLayout: {
            template:
              '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: { template: "<div><slot /><slot name='footer' /></div>" },
          ConfirmDialog: true,
          EmptyState: true,
          Select: true,
          PlatformIcon: true,
          Icon: true,
          GroupRateMultipliersModal: true,
          GroupRPMOverridesModal: true,
          GroupCapacityBadge: true,
          VueDraggable: true,
          Teleport: true,
        },
      },
    });

    await flushPromises();
    await flushPromises();

    const text = wrapper.text();
    expect(text).toContain("¥9.375");
    expect(text).toContain("¥18.75");
    expect(text).toContain("USD 10");
    expect(text).toContain("≈¥75.00");
    expect(text).not.toContain("$1.25");
    expect(text).not.toContain("$2.50");
  });
});
