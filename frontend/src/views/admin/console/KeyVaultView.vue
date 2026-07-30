<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">密钥库</h1>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-outline" type="button" :disabled="loading" @click="reload">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <button v-if="activeTab === 'accounts'" class="btn btn-primary" type="button" data-test="open-create-account" @click="openCreate">
            <Icon name="plus" size="sm" />
            录入 AI 账号
          </button>
          <button v-else class="btn btn-primary" type="button" data-test="open-create-provider" @click="openCreateProvider">
            <Icon name="plus" size="sm" />
            录入视频通道
          </button>
        </div>
      </div>

      <!-- Tab 切换 -->
      <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="rounded-md px-4 py-1.5 text-sm font-medium transition-colors"
          :class="activeTab === tab.key
            ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
            : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- 通用 AI 账号 -->
      <section v-show="activeTab === 'accounts'" class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">名称</th>
                <th class="px-5 py-3 font-medium">平台</th>
                <th class="px-5 py-3 font-medium">分组</th>
                <th class="px-5 py-3 font-medium">接入方式</th>
                <th class="px-5 py-3 font-medium">状态</th>
                <th class="px-5 py-3 font-medium">最近使用</th>
                <th class="px-5 py-3 font-medium">备注</th>
                <th class="px-5 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="account in accounts" :key="account.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-5 py-3 font-medium text-gray-900 dark:text-white">{{ account.name }}</td>
                <td class="px-5 py-3 text-gray-500 dark:text-gray-400">{{ platformLabel(account.platform) }}</td>
                <td class="px-5 py-3">
                  <AccountGroupsCell :groups="account.groups" />
                </td>
                <td class="px-5 py-3 text-gray-600 dark:text-gray-300">{{ accountTypeLabel(account.type) }}</td>
                <td class="px-5 py-3">
                  <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="accountStatusClass(account)">
                    {{ accountStatusLabel(account) }}
                  </span>
                </td>
                <td class="px-5 py-3 text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(account.last_used_at) }}</td>
                <td class="max-w-[200px] truncate px-5 py-3 text-xs text-gray-500 dark:text-gray-400" :title="account.notes || ''">
                  {{ account.notes || '—' }}
                </td>
                <td class="px-5 py-3">
                  <div class="flex flex-wrap justify-end gap-1.5">
                    <button
                      class="btn btn-sm btn-outline"
                      type="button"
                      :disabled="accountActionId === account.id"
                      :data-test="`check-account-connectivity-${account.id}`"
                      @click="testAccountConnectivity(account)"
                    >
                      {{ accountActionId === account.id ? '检查中…' : '检查连接' }}
                    </button>
                    <template v-if="account.status === 'error'">
                      <button
                        class="btn btn-sm btn-outline"
                        type="button"
                        :disabled="accountActionId === account.id"
                        @click="clearAccountError(account)"
                      >
                        清除异常
                      </button>
                    </template>
                    <button class="btn btn-sm btn-outline" type="button" @click="openEditAccount(account)">
                      <Icon name="edit" size="sm" />
                      编辑
                    </button>
                    <button
                      class="btn btn-sm btn-outline"
                      type="button"
                      @click="toggleAccountStatus(account)"
                    >
                      {{ account.status === 'active' ? '停用' : '启用' }}
                    </button>
                    <button class="btn btn-sm btn-outline !text-red-600 dark:!text-red-400" type="button" @click="removeAccount(account)">
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && !accounts.length">
                <td colspan="8" class="px-5 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
                  还没有录入任何 AI 账号。点右上角「录入 AI 账号」添加第一把。
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="accountsTotal > accounts.length" class="border-t border-gray-100 px-5 py-3 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
          共 {{ accountsTotal }} 个账号，当前展示前 {{ accounts.length }} 个。
        </div>
      </section>

      <!-- 视频通道：页内完整管理（含一次性真实调用授权门禁，不离开密钥库） -->
      <section v-show="activeTab === 'video'" class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">名称</th>
                <th class="px-5 py-3 font-medium">平台</th>
                <th class="px-5 py-3 font-medium">分组</th>
                <th class="px-5 py-3 font-medium">状态</th>
                <th class="px-5 py-3 font-medium">最近使用</th>
                <th class="px-5 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="provider in providers" :key="provider.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-5 py-3 font-medium text-gray-900 dark:text-white">
                  {{ provider.display_name }}
                </td>
                <td class="px-5 py-3 text-gray-600 dark:text-gray-300">{{ videoPlatformLabel(provider.provider) }}</td>
                <td class="px-5 py-3 text-gray-600 dark:text-gray-300">{{ provider.group_name || '—' }}</td>
                <td class="px-5 py-3">
                  <span
                    class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                    :class="provider.enabled ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                  >
                    {{ provider.enabled ? '启用中' : '已停用' }}
                  </span>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ provider.api_key_configured ? `凭证已配置 ${provider.masked_key || ''}` : '凭证未配置' }}
                  </div>
                </td>
                <td class="px-5 py-3 text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(provider.tiny_real_consumed_at) }}</td>
                <td class="px-5 py-3">
                  <div class="flex flex-wrap justify-end gap-1.5">
                    <button
                      class="btn btn-sm btn-outline"
                      type="button"
                      :disabled="providerActionId === provider.id"
                      :data-test="`check-provider-connectivity-${provider.id}`"
                      @click="testProviderConnectivity(provider)"
                    >
                      {{ providerActionId === provider.id ? '检查中…' : '检查连接' }}
                    </button>
                    <button
                      class="btn btn-sm btn-outline"
                      type="button"
                      :data-test="`edit-provider-${provider.id}`"
                      @click="openEditProvider(provider)"
                    >
                      <Icon name="edit" size="sm" />
                      编辑
                    </button>
                    <button
                      v-if="needsVideoAuth(provider)"
                      class="btn btn-sm btn-outline"
                      type="button"
                      :disabled="providerActionId === provider.id"
                      :data-test="`retry-auth-provider-${provider.id}`"
                      @click="retryProviderAuth(provider)"
                    >
                      重试授权
                    </button>
                    <button
                      class="btn btn-sm btn-outline"
                      type="button"
                      :data-test="`toggle-provider-${provider.id}`"
                      @click="toggleProvider(provider)"
                    >
                      {{ provider.enabled ? '停用' : '启用' }}
                    </button>
                    <button
                      class="btn btn-sm btn-outline !text-red-600 dark:!text-red-400"
                      type="button"
                      :data-test="`remove-provider-${provider.id}`"
                      @click="removeProvider(provider)"
                    >
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && !providers.length">
                <td colspan="6" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  还没有视频通道。点右上角「录入视频通道」接入第一家。
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- 账号 新增/编辑 弹窗 -->
      <div v-if="accountModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeAccountModal">
        <div class="w-full max-w-lg rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ editingAccount ? '编辑 AI 账号' : '录入 AI 账号' }}
          </h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            只需要填平台、名称和密钥。密钥保存后不再回显，留空表示保留当前密钥。
          </p>
          <form class="mt-5 space-y-4" data-test="account-form" @submit.prevent="saveAccount">
            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">平台</label>
                <select v-model="accountForm.platform" class="input" data-test="account-platform" :disabled="!!editingAccount">
                  <option value="anthropic">Claude（Anthropic）</option>
                  <option value="openai">OpenAI</option>
                  <option value="gemini">Gemini</option>
                  <option value="antigravity">Antigravity</option>
                  <option value="hc_atom">HC-ATOM（图片）</option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">名称</label>
                <input v-model="accountForm.name" class="input" maxlength="100" placeholder="例如：老板的 Claude 主账号" data-test="account-name" required />
              </div>
            </div>
            <div>
              <GroupSelector
                v-model="accountForm.groupIds"
                :groups="platformGroups"
                :platform="accountForm.platform"
                data-test="account-groups"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                必选，至少一个分组。账号的调用按分组计费与路由（例如作图走 media 组）。
              </p>
              <div
                v-if="!platformGroups.length"
                class="mt-2 border-t border-gray-200 pt-3 text-xs dark:border-dark-700"
                data-test="group-quick-create"
              >
                <p class="text-gray-600 dark:text-gray-300">当前平台还没有分组，先建一个（也可到「模型分组」页精细调整）：</p>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                  <button class="btn btn-sm btn-outline" type="button" :disabled="creatingGroup" data-test="quick-create-media" @click="quickCreateGroup(isHCAtomAccount ? 'HC-ATOM 图片组' : `${accountForm.platform}-图片组`, accountForm.platform, 'account', 'media')">
                    {{ isHCAtomAccount ? '创建 HC-ATOM 图片组' : '创建图片组' }}
                  </button>
                  <input
                    v-model="quickGroupName"
                    class="input !w-40 !py-1 text-xs"
                    maxlength="50"
                    placeholder="或输入分组名"
                    data-test="quick-create-name"
                  />
                  <button
                    class="btn btn-sm btn-primary"
                    type="button"
                    :disabled="creatingGroup || !quickGroupName.trim()"
                    data-test="quick-create-custom"
                    @click="quickCreateGroup(quickGroupName)"
                  >
                    {{ creatingGroup ? '创建中…' : '创建并选中' }}
                  </button>
                  <RouterLink class="btn btn-sm btn-outline" to="/admin/groups">管理/删除分组</RouterLink>
                </div>
              </div>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">API Key</label>
              <input
                v-model="accountForm.apiKey"
                class="input"
                type="password"
                autocomplete="off"
                maxlength="4000"
                data-test="account-api-key"
                :placeholder="editingAccount ? '留空表示保留当前密钥' : isHCAtomAccount ? 'HC-ATOM API Key' : 'sk-...'"
                :required="!editingAccount"
              />
              <p v-if="isHCAtomAccount" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                密钥使用 HC-ATOM 独立加密域保存，管理接口不会回显明文。
              </p>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ isHCAtomAccount ? '固定接口地址' : '接口地址（可选）' }}
              </label>
              <p
                v-if="isHCAtomAccount"
                class="rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:bg-gray-800/60 dark:text-gray-300"
                data-test="hc-atom-base-url-locked"
              >
                同步图片：{{ HC_ATOM_SYNC_IMAGE_BASE_URL }}/v1/images/generations<br />
                异步图片：{{ HC_ATOM_ASYNC_IMAGE_BASE_URL }}/image/generation/tasks
              </p>
              <input
                v-else
                v-model="accountForm.baseUrl"
                class="input"
                maxlength="500"
                placeholder="留空使用官方默认地址"
                data-test="account-base-url"
              />
              <p v-if="isHCAtomAccount" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                系统按模型自动选择同步或异步协议，不需要手工填写地址。
              </p>
            </div>
            <div
              v-if="isHCAtomAccount"
              class="rounded-lg border border-cyan-200 bg-cyan-50 p-4 text-sm dark:border-cyan-900 dark:bg-cyan-950/30"
              data-test="hc-atom-model-directory"
            >
              <p class="font-medium text-cyan-900 dark:text-cyan-200">本次已授权并启用的 5 个图片模型</p>
              <ul class="mt-2 list-disc pl-5 text-cyan-800 dark:text-cyan-300">
                <li v-for="model in HC_ATOM_MEDIA_ENABLED_MODELS" :key="model">{{ model }}</li>
              </ul>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">备注（可选）</label>
              <input v-model="accountForm.notes" class="input" maxlength="200" placeholder="例如：包月订阅，到期 8 月底" />
            </div>
            <div class="flex justify-end gap-2 pt-2">
              <button class="btn btn-outline" type="button" data-test="cancel-account" @click="closeAccountModal">取消</button>
              <button class="btn btn-primary" type="submit" data-test="save-account" :disabled="savingAccount">
                <Icon name="check" size="sm" />
                保存
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- 视频通道 录入弹窗（字段顺序对齐 LLM 账号表单） -->
      <div v-if="providerModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeProviderModal">
        <div class="w-full max-w-lg rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">录入视频通道</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            选平台、起个名、选分组、填密钥。密钥保存后不再回显。
          </p>
          <form class="mt-5 space-y-4" data-test="provider-form" @submit.prevent="saveProvider">
            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">平台</label>
                <select v-model="providerForm.provider" class="input" data-test="provider-platform">
                  <option
                    v-for="platform in videoPlatforms"
                    :key="platform.provider"
                    :value="platform.provider"
                    :disabled="!platform.adapter_ready"
                  >
                    {{ platform.display_name }}{{ platform.adapter_ready ? '' : '（即将接入）' }}
                  </option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">名称</label>
                <input v-model="providerForm.name" class="input" maxlength="100" placeholder="哪家的哪个号，随便打" data-test="provider-name" required />
              </div>
            </div>
            <div>
              <GroupSelector
                v-model="providerForm.groupIds"
                :groups="providerPlatformGroups"
                data-test="provider-groups"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                必选，通道的调用按分组计费与路由。
              </p>
              <div
                v-if="!providerPlatformGroups.length"
                class="mt-2 border-t border-gray-200 pt-3 text-xs dark:border-dark-700"
                data-test="provider-group-quick-create"
              >
                <p class="text-gray-600 dark:text-gray-300">还没有分组，先建一个（也可到「模型分组」页精细调整）：</p>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                  <button class="btn btn-sm btn-outline" type="button" :disabled="creatingGroup" data-test="provider-quick-create-video" @click="quickCreateGroup(defaultVideoGroupName(providerForm.provider), videoProviderGroupPlatform(providerForm.provider), 'provider', 'video')">
                    {{ defaultVideoGroupName(providerForm.provider) }}
                  </button>
                  <input
                    v-model="quickGroupName"
                    class="input !w-40 !py-1 text-xs"
                    maxlength="50"
                    placeholder="或输入分组名"
                    data-test="provider-quick-create-name"
                  />
                  <button
                    class="btn btn-sm btn-primary"
                    type="button"
                    :disabled="creatingGroup || !quickGroupName.trim()"
                    data-test="provider-quick-create-custom"
                    @click="quickCreateGroup(quickGroupName, videoProviderGroupPlatform(providerForm.provider), 'provider', 'video')"
                  >
                    {{ creatingGroup ? '创建中…' : '创建并选中' }}
                  </button>
                  <RouterLink class="btn btn-sm btn-outline" to="/admin/groups">管理/删除分组</RouterLink>
                </div>
              </div>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">API Key</label>
              <input
                v-model="providerForm.apiKey"
                class="input"
                type="password"
                autocomplete="off"
                maxlength="4000"
                data-test="provider-api-key"
                placeholder="上游平台发放的密钥"
                required
              />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">接口地址</label>
              <p class="rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-600 dark:bg-gray-800/60 dark:text-gray-300" data-test="provider-base-url-locked">
                {{ selectedVideoPlatform?.default_base_url || '官方默认地址' }}
              </p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ isSelectedHCAtomProvider
                  ? 'HC-ATOM 中转地址由协议固定，不允许自定义。'
                  : '当前仅支持官方直连；自定义中转尚未打通授权与调度，暂不可填。' }}
              </p>
            </div>
            <div class="flex flex-col gap-2">
              <label class="flex min-h-6 items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="providerForm.enabled" type="checkbox" data-test="provider-enabled" />
                保存后启用
              </label>
              <label
                v-if="supportsTinyRealAuthorization"
                class="flex min-h-6 items-center gap-2 text-sm text-gray-700 dark:text-gray-300"
              >
                <input v-model="providerForm.authorizeAfterSave" type="checkbox" data-test="provider-authorize-after-save" />
                保存后自动授权一次最小真实调用
              </label>
              <p v-if="supportsTinyRealAuthorization" class="text-xs text-gray-500 dark:text-gray-400">
                默认关闭；只有明确勾选才记录授权，保存配置本身不会触发上游调用。
              </p>
              <p v-else class="text-xs text-gray-500 dark:text-gray-400" data-test="provider-hc-dispatch-note">
                HC-ATOM 通道保存后即可按分组调度；保存配置不会触发真实视频任务。
              </p>
            </div>
            <div class="flex justify-end gap-2 pt-2">
              <button class="btn btn-outline" type="button" data-test="cancel-provider" @click="closeProviderModal">取消</button>
              <button class="btn btn-primary" type="submit" data-test="save-provider" :disabled="savingProvider">
                <Icon name="check" size="sm" />
                保存
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- 视频通道 编辑弹窗 -->
      <div v-if="editProviderModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeEditProviderModal">
        <div class="w-full max-w-lg rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">编辑视频通道</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            密钥留空表示保留当前密钥；平台不可更改。
          </p>
          <form class="mt-5 space-y-4" data-test="edit-provider-form" @submit.prevent="saveEditProvider">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">名称</label>
              <input v-model="editProviderForm.displayName" class="input" maxlength="100" required />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">分组</label>
              <select v-model="editProviderForm.groupId" class="input" data-test="edit-provider-group" required>
                <option v-for="group in editProviderGroups" :key="group.id" :value="group.id">{{ group.name }}</option>
              </select>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">API Key</label>
              <input
                v-model="editProviderForm.apiKey"
                class="input"
                type="password"
                autocomplete="off"
                maxlength="4000"
                placeholder="留空表示保留当前密钥"
              />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">接口地址</label>
              <p
                v-if="isEditingHCAtomProvider"
                class="rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-600 dark:bg-gray-800/60 dark:text-gray-300"
                data-test="edit-provider-base-url-locked"
              >
                {{ editingVideoPlatform?.default_base_url || 'HC-ATOM 固定地址' }}
              </p>
              <input
                v-else
                v-model="editProviderForm.baseUrl"
                class="input"
                maxlength="500"
                placeholder="留空使用官方默认地址"
                data-test="edit-provider-base-url"
              />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">默认模型</label>
              <p
                v-if="isEditingHCAtomProvider"
                class="rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-600 dark:bg-gray-800/60 dark:text-gray-300"
                data-test="edit-provider-default-model-locked"
              >
                {{ editingVideoPlatform?.default_model || '系统按通道自动匹配' }}
              </p>
              <input
                v-else
                v-model="editProviderForm.defaultModel"
                class="input"
                maxlength="200"
                placeholder="留空使用平台默认"
                data-test="edit-provider-default-model"
              />
              <p v-if="isEditingHCAtomProvider" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                HC-ATOM 的接口和模型由通道协议固定，保存时会自动清理旧配置，不需要手工填写。
              </p>
            </div>
            <div class="flex justify-end gap-2 pt-2">
              <button class="btn btn-outline" type="button" data-test="cancel-edit-provider" @click="closeEditProviderModal">取消</button>
              <button class="btn btn-primary" type="submit" data-test="save-edit-provider" :disabled="savingEditProvider">
                <Icon name="check" size="sm" />
                保存
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import {
  HC_ATOM_ASYNC_IMAGE_BASE_URL,
  HC_ATOM_IMAGE_BASE_URL,
  HC_ATOM_MEDIA_ENABLED_MODELS,
  HC_ATOM_MEDIA_GROUP_PRICES,
  HC_ATOM_SYNC_IMAGE_BASE_URL,
  HC_ATOM_VIDEO_ENABLED_MODELS,
  HC_ATOM_VIDEO_GROUP_PRICES,
  HC_ATOM_VIDEO_V1_MODEL,
  HC_ATOM_VIDEO_V3_MODEL,
  buildHCAtomImageCredentials,
  isHCAtomImageGroup,
  isHCAtomVideoGroup,
} from '@/components/account/hcAtomAdminContract'
import { adminAPI } from '@/api/admin'
import type { Account, AccountPlatform, AdminGroup, GroupPlatform } from '@/types'
import type { VideoPlatformContract, VideoProviderAccount, VideoProviderContract } from '@/api/admin/video'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { createIdempotencyKey } from '@/utils/idempotencyKey'
import { requestConfirmation } from '@/composables/useAppDialog'
import { CONSOLE_ERROR_ZH, formatDateTime } from './consoleUtils'

const appStore = useAppStore()
const route = useRoute()

type TabKey = 'accounts' | 'video'
const tabs: Array<{ key: TabKey; label: string }> = [
  { key: 'accounts', label: '通用 AI 账号（文字 / 作图）' },
  { key: 'video', label: '视频通道' },
]
const activeTab = ref<TabKey>('accounts')

const loading = ref(false)
const accounts = ref<Account[]>([])
const accountsTotal = ref(0)
const providers = ref<VideoProviderAccount[]>([])
const groups = ref<AdminGroup[]>([])
const accountActionId = ref<number | null>(null)
const providerActionId = ref<number | null>(null)

// ---- 通用 AI 账号 ----

const accountModalOpen = ref(false)
const editingAccount = ref<Account | null>(null)
const savingAccount = ref(false)
const accountCreateIdempotencyKey = ref<string | null>(null)
const groupsLoadFailed = ref(false)

const accountForm = reactive({
  platform: 'anthropic' as AccountPlatform,
  name: '',
  apiKey: '',
  baseUrl: '',
  notes: '',
  groupIds: [] as number[],
})
const isHCAtomAccount = computed(() => accountForm.platform === 'hc_atom')

// 分组必选：当前平台可选的分组；切平台时清掉不再匹配的已选 id
const platformGroups = computed(() => groups.value.filter((group) => {
  if (group.platform !== accountForm.platform || group.status !== 'active') return false
  if (accountForm.platform === 'hc_atom') return isHCAtomImageGroup(group)
  return true
}))
const creatingGroup = ref(false)
const quickGroupName = ref('')

watch(
  () => accountForm.platform,
  () => {
    const valid = new Set(platformGroups.value.map((g) => g.id))
    accountForm.groupIds = accountForm.groupIds.filter((id) => valid.has(id))
  },
)

async function quickCreateGroup(
  name: string,
  platform: GroupPlatform = accountForm.platform,
  target: 'account' | 'provider' = 'account',
  capability?: 'media' | 'video',
) {
  const trimmed = name.trim()
  if (!trimmed || creatingGroup.value) return
  const isHCAtom = platform === 'hc_atom'
  const isMedia = capability
    ? capability === 'media'
    : target === 'account' && (isHCAtom || trimmed === 'media')
  const videoModels = target === 'provider'
    ? hcAtomVideoModelsForProvider(providerForm.provider)
    : HC_ATOM_VIDEO_ENABLED_MODELS
  const models = isHCAtom
    ? [...(isMedia ? HC_ATOM_MEDIA_ENABLED_MODELS : videoModels)]
    : []
  const existing = groups.value.find((group) => group.name === trimmed)
  if (existing) {
    const reusable = target === 'account'
      ? existing.platform === platform
        && existing.status === 'active'
        && (!isHCAtom || (isMedia && isHCAtomImageGroup(existing)))
      : isEligibleVideoProviderGroup(existing, providerForm.provider)
    if (reusable) {
      if (target === 'provider') {
        providerForm.groupIds = [existing.id]
      } else if (!accountForm.groupIds.includes(existing.id)) {
        accountForm.groupIds = [...accountForm.groupIds, existing.id]
      }
      quickGroupName.value = ''
      appStore.showSuccess(`分组「${existing.name}」已存在，已直接选中`)
      return
    }
    appStore.showError(
      existing.status !== 'active'
        ? `分组名「${trimmed}」已被停用分组占用，请先到「模型分组」启用或改名`
        : `分组名「${trimmed}」已被其他平台或能力占用，请换一个明确的名称`,
    )
    return
  }
  creatingGroup.value = true
  try {
    // 后端建组契约要求全量字段（缺失会 500）；与 GroupsView 默认表单对齐，按模板名预置媒体开关
    const created = await adminAPI.groups.create({
      name: trimmed,
      description: '',
      platform,
      rate_multiplier: 1,
      is_exclusive: false,
      subscription_type: 'standard',
      daily_limit_usd: null,
      weekly_limit_usd: null,
      monthly_limit_usd: null,
      allow_image_generation: isMedia,
      allow_batch_image_generation: isHCAtom && isMedia,
      image_rate_independent: false,
      image_rate_multiplier: 1,
      batch_image_discount_multiplier: 0.5,
      batch_image_hold_multiplier: 0.6,
      image_price_1k: isHCAtom && isMedia ? HC_ATOM_MEDIA_GROUP_PRICES.image_price_1k : null,
      image_price_2k: isHCAtom && isMedia ? HC_ATOM_MEDIA_GROUP_PRICES.image_price_2k : null,
      image_price_4k: isHCAtom && isMedia ? HC_ATOM_MEDIA_GROUP_PRICES.image_price_4k : null,
      video_rate_independent: false,
      video_rate_multiplier: 1,
      video_price_480p: isHCAtom && !isMedia ? HC_ATOM_VIDEO_GROUP_PRICES.video_price_480p : null,
      video_price_720p: isHCAtom && !isMedia ? HC_ATOM_VIDEO_GROUP_PRICES.video_price_720p : null,
      video_price_1080p: isHCAtom && !isMedia ? HC_ATOM_VIDEO_GROUP_PRICES.video_price_1080p : null,
      peak_rate_enabled: false,
      peak_start: '',
      peak_end: '',
      peak_rate_multiplier: 1,
      claude_code_only: false,
      fallback_group_id: null,
      fallback_group_id_on_invalid_request: null,
      allow_messages_dispatch: false,
      require_oauth_only: false,
      require_privacy_set: false,
      models_list_config: { enabled: isHCAtom, models },
      model_routing_enabled: false,
      supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'],
      mcp_xml_inject: true,
      copy_accounts_from_group_ids: [],
      rpm_limit: 0,
    })
    await loadGroups()
    if (target === 'provider') {
      if (!providerForm.groupIds.includes(created.id)) {
        providerForm.groupIds = [...providerForm.groupIds, created.id]
      }
    } else if (!accountForm.groupIds.includes(created.id)) {
      accountForm.groupIds = [...accountForm.groupIds, created.id]
    }
    quickGroupName.value = ''
    appStore.showSuccess(`分组「${created.name}」已创建并选中`)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '创建分组失败', CONSOLE_ERROR_ZH))
  } finally {
    creatingGroup.value = false
  }
}

const platformDefaults: Record<string, string> = {
  anthropic: 'https://api.anthropic.com',
  openai: 'https://api.openai.com',
  gemini: 'https://generativelanguage.googleapis.com',
  antigravity: '',
  hc_atom: HC_ATOM_IMAGE_BASE_URL,
}

function platformLabel(platform: string): string {
  const labels: Record<string, string> = {
    anthropic: 'Claude',
    openai: 'OpenAI',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
    hc_atom: 'HC-ATOM',
  }
  return labels[platform] || platform
}

function accountTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    apikey: 'API Key',
    oauth: 'OAuth 登录',
    'setup-token': 'Setup Token',
    upstream: '上游中转',
    bedrock: 'AWS Bedrock',
    service_account: '服务账号',
  }
  return labels[type] || type
}

function accountStatusLabel(account: Account): string {
  if (account.status === 'error') return '异常'
  if (account.status === 'inactive') return '已停用'
  return '正常'
}

function accountStatusClass(account: Account): string {
  if (account.status === 'error') return 'bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300'
  if (account.status === 'inactive') return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
}

function openCreate() {
  editingAccount.value = null
  accountForm.platform = 'anthropic'
  accountForm.name = ''
  accountForm.apiKey = ''
  accountForm.baseUrl = ''
  accountForm.notes = ''
  accountForm.groupIds = []
  quickGroupName.value = ''
  accountCreateIdempotencyKey.value = createIdempotencyKey()
  accountModalOpen.value = true
}

function openEditAccount(account: Account) {
  editingAccount.value = account
  accountForm.platform = account.platform
  accountForm.name = account.name
  accountForm.apiKey = ''
  accountForm.baseUrl = account.platform === 'hc_atom'
    ? HC_ATOM_IMAGE_BASE_URL
    : String(account.credentials?.base_url ?? '')
  accountForm.notes = account.notes || ''
  // 回填已绑定分组；老数据若没有 group_ids 字段则用预加载的 groups 兜底
  accountForm.groupIds = [...(account.group_ids ?? account.groups?.map((g) => g.id) ?? [])]
  quickGroupName.value = ''
  accountModalOpen.value = true
}

function clearAccountSecrets() {
  // 密钥永远不落地在可回显的 reactive 状态里：取消或保存后都要立即清空，
  // 不依赖弹窗卸载时机（v-if 卸载是异步的，明文会在这段时间内留在内存/响应式图里）。
  accountForm.apiKey = ''
}

function closeAccountModal() {
  accountModalOpen.value = false
  editingAccount.value = null
  accountCreateIdempotencyKey.value = null
  clearAccountSecrets()
}

async function saveAccount() {
  // 分组必选：未选组直接拦截并显式报错，避免账号落成「无组孤儿」（计费/路由都走不通）
  if (accountForm.groupIds.length === 0) {
    appStore.showError('请至少选择一个图片分组；没有可用分组时先点下方「创建 HC-ATOM 图片组」')
    return
  }
  if (groupsLoadFailed.value) {
    appStore.showError('分组加载失败，无法安全保存账号。请刷新页面后重试')
    return
  }
  if (
    accountForm.platform === 'hc_atom'
    && accountForm.groupIds.some((id) => !platformGroups.value.some((group) => group.id === id))
  ) {
    appStore.showError('请选择启用状态且只包含已授权 5 个图片模型的 HC-ATOM 图片组')
    return
  }
  savingAccount.value = true
  try {
    if (editingAccount.value) {
      let credentials: Record<string, unknown>
      if (accountForm.platform === 'hc_atom') {
        credentials = accountForm.apiKey.trim()
          ? buildHCAtomImageCredentials(accountForm.apiKey)
          : {
              protocol: 'hc_atom',
              model_mapping: Object.fromEntries(
                HC_ATOM_MEDIA_ENABLED_MODELS.map((model) => [model, model]),
              ),
            }
      } else {
        credentials = { ...(editingAccount.value.credentials || {}) }
        if (accountForm.apiKey.trim()) credentials.api_key = accountForm.apiKey.trim()
        if (accountForm.baseUrl.trim()) {
          credentials.base_url = accountForm.baseUrl.trim()
        } else if (platformDefaults[accountForm.platform]) {
          credentials.base_url = credentials.base_url || platformDefaults[accountForm.platform]
        }
      }
      await adminAPI.accounts.update(editingAccount.value.id, {
        name: accountForm.name.trim(),
        notes: accountForm.notes.trim() || null,
        credentials,
        group_ids: [...accountForm.groupIds],
      })
      appStore.showSuccess('账号已更新')
    } else {
      const credentials: Record<string, unknown> = accountForm.platform === 'hc_atom'
        ? buildHCAtomImageCredentials(accountForm.apiKey)
        : { api_key: accountForm.apiKey.trim() }
      if (accountForm.platform !== 'hc_atom') {
        const baseUrl = accountForm.baseUrl.trim() || platformDefaults[accountForm.platform]
        if (baseUrl) credentials.base_url = baseUrl
      }
      await adminAPI.accounts.create({
        name: accountForm.name.trim(),
        notes: accountForm.notes.trim() || null,
        platform: accountForm.platform,
        type: 'apikey',
        credentials,
        group_ids: [...accountForm.groupIds],
      }, accountCreateIdempotencyKey.value ?? createIdempotencyKey())
      appStore.showSuccess('账号已录入密钥库')
    }
    closeAccountModal()
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存账号失败', CONSOLE_ERROR_ZH))
  } finally {
    savingAccount.value = false
  }
}

async function toggleAccountStatus(account: Account) {
  const next = account.status === 'active' ? 'inactive' : 'active'
  if (next === 'inactive') {
    const confirmed = await requestConfirmation({
      message: `确定停用账号「${account.name}」？停用后走该账号的调用会立即失败。`,
      danger: true,
    })
    if (!confirmed) return
  }
  try {
    await adminAPI.accounts.toggleStatus(account.id, next)
    appStore.showSuccess(next === 'active' ? '账号已启用' : '账号已停用')
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '切换账号状态失败', CONSOLE_ERROR_ZH))
  }
}

async function removeAccount(account: Account) {
  const confirmed = await requestConfirmation({
    message: `确定删除账号「${account.name}」？删除后走该账号的调用会失败。`,
    danger: true,
  })
  if (!confirmed) return
  try {
    await adminAPI.accounts.delete(account.id)
    appStore.showSuccess('账号已删除')
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除账号失败', CONSOLE_ERROR_ZH))
  }
}

async function testAccountConnectivity(account: Account) {
  accountActionId.value = account.id
  try {
    const result = await adminAPI.accounts.checkConnectivity(account.id)
    showConnectivityResult(result)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '连接检查失败', CONSOLE_ERROR_ZH))
  } finally {
    accountActionId.value = null
  }
}

function showConnectivityResult(result: {
  status: 'ok' | 'warning' | 'error'
  message: string
  latency_ms?: number
  generation_started: boolean
}) {
  const latency = typeof result.latency_ms === 'number' ? `（${result.latency_ms}ms）` : ''
  const baseMessage = result.message || '连接检查已完成'
  const generation = result.generation_started || baseMessage.includes('未创建生成任务')
    ? ''
    : '；未创建生成任务'
  const message = `${baseMessage}${latency}${generation}`
  if (result.status === 'ok') {
    appStore.showSuccess(message, 5000)
  } else if (result.status === 'warning') {
    appStore.showWarning(message, 6000)
  } else {
    appStore.showError(message, 6000)
  }
}

async function clearAccountError(account: Account) {
  accountActionId.value = account.id
  try {
    let updated = await adminAPI.accounts.clearError(account.id)
    if (updated.status === 'error') {
      updated = await adminAPI.accounts.recoverState(account.id)
    }
    if (updated.status === 'error') {
      appStore.showError('已尝试清除异常，但账号仍处于异常状态，请检查密钥或上游连通性')
    } else {
      appStore.showSuccess('异常状态已清除')
    }
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '清除异常失败', CONSOLE_ERROR_ZH))
  } finally {
    accountActionId.value = null
  }
}

async function loadAccounts() {
  const res = await adminAPI.accounts.list(1, 100)
  accounts.value = res.items || []
  accountsTotal.value = res.total || accounts.value.length
}

async function loadGroups() {
  try {
    groups.value = await adminAPI.groups.getAllIncludingInactive()
    groupsLoadFailed.value = false
  } catch (error) {
    groups.value = []
    groupsLoadFailed.value = true
    throw error
  }
}

// ---- 视频通道（页内完整管理：列表/启停/删除/录入 + 保存后自动授权） ----

function videoProviderGroupPlatform(provider: string): GroupPlatform {
  return provider.startsWith('hc_atom_') ? 'hc_atom' : 'openai'
}

function defaultVideoGroupName(provider: string): string {
  return provider.startsWith('hc_atom_') ? 'HC-ATOM 视频组' : '视频组'
}

function hcAtomVideoModelsForProvider(provider: string): readonly string[] {
  if (provider === 'hc_atom_video_v1') return [HC_ATOM_VIDEO_V1_MODEL]
  if (provider === 'hc_atom_seedance_v3') return [HC_ATOM_VIDEO_V3_MODEL]
  return HC_ATOM_VIDEO_ENABLED_MODELS
}

const providerModalOpen = ref(false)
const savingProvider = ref(false)
const editProviderModalOpen = ref(false)
const savingEditProvider = ref(false)
const editingProvider = ref<VideoProviderAccount | null>(null)
const videoContract = ref<VideoProviderContract | null>(null)

const editProviderForm = reactive({
  displayName: '',
  apiKey: '',
  baseUrl: '',
  defaultModel: '',
  groupId: 0,
})

const providerForm = reactive({
  provider: 'seedance',
  name: '',
  apiKey: '',
  baseUrl: '',
  groupIds: [] as number[],
  enabled: true,
  authorizeAfterSave: false,
})

function isEligibleVideoProviderGroup(group: AdminGroup, provider: string): boolean {
  if (
    group.status !== 'active'
    || group.platform !== videoProviderGroupPlatform(provider)
  ) return false
  if (!provider.startsWith('hc_atom_')) return true
  if (!isHCAtomVideoGroup(group)) return false
  const configured = new Set(
    group.models_list_config?.enabled
      ? group.models_list_config.models.map((model) => model.trim()).filter(Boolean)
      : [],
  )
  return hcAtomVideoModelsForProvider(provider).every((model) => configured.has(model))
}

const providerPlatformGroups = computed(() => (
  groups.value.filter((group) => isEligibleVideoProviderGroup(group, providerForm.provider))
))

watch(
  () => providerForm.provider,
  () => {
    providerForm.groupIds = providerForm.groupIds.filter((id) => (
      providerPlatformGroups.value.some((group) => group.id === id)
    ))
    if (providerForm.provider !== 'seedance') {
      providerForm.authorizeAfterSave = false
    }
  },
)

const editProviderGroups = computed(() => (
  editingProvider.value
    ? groups.value.filter((group) => isEligibleVideoProviderGroup(group, editingProvider.value!.provider))
    : []
))

const isEditingHCAtomProvider = computed(
  () => editingProvider.value?.provider.startsWith('hc_atom_') ?? false,
)

// contract 请求失败时回落为只含 seedance 的静态兜底平台
const FALLBACK_VIDEO_PLATFORMS: VideoPlatformContract[] = [
  { provider: 'seedance', display_name: 'Seedance', default_base_url: '', default_model: '', adapter_ready: true },
]

const videoPlatforms = computed<VideoPlatformContract[]>(() => {
  const platforms = videoContract.value?.platforms
  return platforms && platforms.length ? platforms : FALLBACK_VIDEO_PLATFORMS
})

const editingVideoPlatform = computed(
  () => videoPlatforms.value.find((platform) => platform.provider === editingProvider.value?.provider) ?? null,
)

const selectedVideoPlatform = computed(
  () => videoPlatforms.value.find((platform) => platform.provider === providerForm.provider) ?? null,
)

const isSelectedHCAtomProvider = computed(
  () => providerForm.provider.startsWith('hc_atom_'),
)

const supportsTinyRealAuthorization = computed(
  () => providerForm.provider === 'seedance',
)

const STATIC_VIDEO_PLATFORM_LABELS: Record<string, string> = {
  seedance: 'Seedance',
  jimeng: '即梦',
  veo: 'Veo 3.1',
  kling: '快乐小马',
}

function videoPlatformLabel(provider: string): string {
  return (
    videoPlatforms.value.find((platform) => platform.provider === provider)?.display_name
    || STATIC_VIDEO_PLATFORM_LABELS[provider]
    || provider
  )
}

async function loadVideoContract() {
  try {
    videoContract.value = await adminAPI.video.contract()
  } catch {
    // 平台目录加载失败不阻塞列表：下拉回落为 seedance 静态兜底
    videoContract.value = null
  }
}

function openCreateProvider() {
  providerForm.provider = videoPlatforms.value.find((platform) => platform.adapter_ready)?.provider ?? 'seedance'
  providerForm.name = ''
  providerForm.apiKey = ''
  providerForm.baseUrl = ''
  providerForm.groupIds = []
  providerForm.enabled = true
  providerForm.authorizeAfterSave = false
  quickGroupName.value = ''
  providerModalOpen.value = true
  if (!videoContract.value) void loadVideoContract()
}

function clearProviderSecrets() {
  // 与 LLM 账号同一约定：密钥不留在可回显的 reactive 状态里
  providerForm.apiKey = ''
}

function closeProviderModal() {
  providerModalOpen.value = false
  clearProviderSecrets()
}

async function saveProvider() {
  if (providerForm.groupIds.length === 0) {
    appStore.showError('请至少选择一个匹配当前视频协议的分组；没有可用分组时先在下方创建')
    return
  }
  if (groupsLoadFailed.value) {
    appStore.showError('分组加载失败，无法安全保存视频通道。请刷新页面后重试')
    return
  }
  if (providerForm.groupIds.some((id) => !providerPlatformGroups.value.some((group) => group.id === id))) {
    appStore.showError('所选分组与当前视频协议不匹配，请重新选择')
    return
  }
  const platform = selectedVideoPlatform.value
  if (platform && !platform.adapter_ready) {
    appStore.showError(`平台「${platform.display_name}」即将接入，暂时不能录入`)
    return
  }
  savingProvider.value = true
  try {
    const created = await adminAPI.video.createProvider({
      group_id: providerForm.groupIds[0],
      provider: providerForm.provider,
      display_name: providerForm.name.trim(),
      api_key: providerForm.apiKey.trim(),
      enabled: providerForm.enabled,
      // 自定义中转 base_url 尚未打通授权/调度；只允许官方默认（后端亦会拒绝覆盖）
      ...(platform?.default_model ? { default_model: platform.default_model } : {}),
    })
    const wantAuthorize = providerForm.provider === 'seedance' && providerForm.authorizeAfterSave
    closeProviderModal()
    await loadProviders()
    if (wantAuthorize && created.enabled && created.api_key_configured && !created.tiny_real_authorized_at && !created.tiny_real_consumed_at) {
      try {
        await adminAPI.video.authorizeTinyReal(created.id)
        appStore.showSuccess('通道已保存并记录单次授权；等第一次真实出片后通道永久可用')
        await loadProviders()
        return
      } catch (authError) {
        appStore.showError(`通道已保存，但自动授权失败：${extractApiErrorMessage(authError, '可在密钥库「视频通道」中重试授权', CONSOLE_ERROR_ZH)}`)
        return
      }
    }
    appStore.showSuccess('通道已录入密钥库')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存视频通道失败', CONSOLE_ERROR_ZH))
  } finally {
    savingProvider.value = false
  }
}

function needsVideoAuth(provider: VideoProviderAccount): boolean {
  return provider.provider === 'seedance'
    && provider.enabled
    && provider.api_key_configured
    && !provider.tiny_real_authorized_at
    && !provider.tiny_real_consumed_at
}

function openEditProvider(provider: VideoProviderAccount) {
  editingProvider.value = provider
  editProviderForm.displayName = provider.display_name
  editProviderForm.apiKey = ''
  const isHCAtom = provider.provider.startsWith('hc_atom_')
  editProviderForm.baseUrl = isHCAtom ? '' : (provider.base_url || '')
  editProviderForm.defaultModel = isHCAtom ? '' : (provider.default_model || '')
  const eligibleGroups = groups.value.filter((group) => isEligibleVideoProviderGroup(group, provider.provider))
  editProviderForm.groupId = eligibleGroups.some((group) => group.id === provider.group_id)
    ? provider.group_id
    : (eligibleGroups[0]?.id ?? 0)
  editProviderModalOpen.value = true
}

function clearEditProviderSecrets() {
  editProviderForm.apiKey = ''
}

function closeEditProviderModal() {
  editProviderModalOpen.value = false
  editingProvider.value = null
  clearEditProviderSecrets()
}

async function saveEditProvider() {
  if (!editingProvider.value) return
  if (!editProviderForm.groupId) {
    appStore.showError('请选择一个分组')
    return
  }
  if (!editProviderGroups.value.some((group) => group.id === Number(editProviderForm.groupId))) {
    appStore.showError('所选分组与当前视频协议不匹配，请重新选择')
    return
  }
  savingEditProvider.value = true
  try {
    const payload: {
      display_name: string
      group_id: number
      base_url?: string
      default_model?: string
      api_key?: string
    } = {
      display_name: editProviderForm.displayName.trim(),
      group_id: editProviderForm.groupId,
    }
    if (isEditingHCAtomProvider.value) {
      // 历史 HC 通道可能残留错误的接口/模型。显式发送空值，让后端恢复当前协议的固定值。
      payload.base_url = ''
      payload.default_model = ''
    } else {
      const baseUrl = editProviderForm.baseUrl.trim()
      if (baseUrl) payload.base_url = baseUrl
      const defaultModel = editProviderForm.defaultModel.trim()
      if (defaultModel) payload.default_model = defaultModel
    }
    if (editProviderForm.apiKey.trim()) payload.api_key = editProviderForm.apiKey.trim()
    await adminAPI.video.updateProvider(editingProvider.value.id, payload)
    appStore.showSuccess('通道已更新')
    closeEditProviderModal()
    await loadProviders()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '更新视频通道失败', CONSOLE_ERROR_ZH))
  } finally {
    savingEditProvider.value = false
  }
}

async function retryProviderAuth(provider: VideoProviderAccount) {
  providerActionId.value = provider.id
  try {
    await adminAPI.video.authorizeTinyReal(provider.id)
    appStore.showSuccess('已记录单次授权；等第一次真实出片后通道永久可用')
    await loadProviders()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '重试授权失败', CONSOLE_ERROR_ZH))
  } finally {
    providerActionId.value = null
  }
}

async function testProviderConnectivity(provider: VideoProviderAccount) {
  providerActionId.value = provider.id
  try {
    const result = await adminAPI.video.checkProviderConnectivity(provider.id)
    showConnectivityResult(result)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '连接检查失败', CONSOLE_ERROR_ZH))
  } finally {
    providerActionId.value = null
  }
}

async function toggleProvider(provider: VideoProviderAccount) {
  if (provider.enabled) {
    const confirmed = await requestConfirmation({
      message: `确定停用通道「${provider.display_name}」？停用后走该通道的视频调用会立即失败。`,
      danger: true,
    })
    if (!confirmed) return
  }
  try {
    await adminAPI.video.updateProvider(provider.id, { enabled: !provider.enabled })
    appStore.showSuccess(provider.enabled ? '通道已停用' : '通道已启用')
    await loadProviders()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '切换通道状态失败', CONSOLE_ERROR_ZH))
  }
}

async function removeProvider(provider: VideoProviderAccount) {
  const confirmed = await requestConfirmation({
    message: `确定删除通道「${provider.display_name}」？删除后走该通道的视频调用会失败。`,
    danger: true,
  })
  if (!confirmed) return
  try {
    await adminAPI.video.deleteProvider(provider.id)
    appStore.showSuccess('通道已删除')
    await loadProviders()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除通道失败', CONSOLE_ERROR_ZH))
  }
}

async function loadProviders() {
  const res = await adminAPI.video.listProviders()
  providers.value = res.items || []
}

// ---- 汇总加载 ----

async function reload() {
  loading.value = true
  try {
    await Promise.all([loadAccounts(), loadProviders(), loadGroups(), loadVideoContract()])
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载密钥库失败', CONSOLE_ERROR_ZH))
  } finally {
    loading.value = false
  }
}

function syncTabFromRoute() {
  if (route.query.tab === 'video') {
    activeTab.value = 'video'
  }
}

watch(() => route.query.tab, syncTabFromRoute)

onMounted(() => {
  syncTabFromRoute()
  void reload()
})
</script>
