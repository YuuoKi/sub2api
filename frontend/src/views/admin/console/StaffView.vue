<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="ui-heading">服务身份与 API 卡片</h1>
          <p class="ui-subheading mt-1">
            为 QCanvas、n8n、脚本和批量生产流程创建不可登录的服务身份，再签发独立 API 卡片。
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-outline" type="button" :disabled="loading" @click="loadStaff">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <button class="btn btn-primary" type="button" data-test="create-service-identity" @click="openCreateServiceIdentity">
            <Icon name="key" size="sm" />
            新增服务身份
          </button>
        </div>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <input
              v-model="search"
              class="input sm:max-w-xs"
              placeholder="按姓名或邮箱搜索"
              @keyup.enter="loadStaff"
            />
          </div>
          <div class="text-xs text-gray-500 dark:text-gray-400">共 {{ filteredUsers.length }} 个服务身份</div>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-700/40 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3 font-medium">成员</th>
                <th class="px-5 py-3 font-medium">备注</th>
                <th class="px-5 py-3 font-medium">状态</th>
                <th class="px-5 py-3 font-medium">今日花费</th>
                <th class="px-5 py-3 font-medium">累计花费</th>
                <th class="px-5 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <template v-for="user in filteredUsers" :key="user.id">
                <tr class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40" data-test="toggle-expand" @click="toggleExpand(user)">
                  <td class="px-5 py-3">
                    <div class="flex items-center gap-3">
                      <span
                        class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gray-100 text-sm font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                      >
                        {{ staffDisplayName(user.username, user.email).slice(0, 1).toUpperCase() }}
                      </span>
                      <div class="min-w-0">
                        <div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
                          {{ staffDisplayName(user.username, user.email) }}
                          <span v-if="user.member_type === 'tool'" class="text-xs text-gray-400 dark:text-gray-500">工具</span>
                          <span v-if="user.role === 'admin'" class="text-xs text-gray-400 dark:text-gray-500">管理员</span>
                        </div>
                        <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ user.email }}</div>
                      </div>
                    </div>
                  </td>
                  <td class="max-w-[180px] truncate px-5 py-3 text-xs text-gray-500 dark:text-gray-400" :title="user.notes || ''">{{ user.notes || '—' }}</td>
                  <td class="px-5 py-3">
                    <span
                      class="inline-flex rounded-md px-2 py-1 text-xs font-medium"
                      :class="user.status === 'active' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                    >
                      {{ user.status === 'active' ? '在用' : '已停用' }}
                    </span>
                  </td>
                  <td class="px-5 py-3 tabular-nums text-gray-700 dark:text-gray-200">{{ formatMoney(usageMap[user.id]?.today_actual_cost, usdCnyRate) }}</td>
                  <td class="px-5 py-3 tabular-nums text-gray-900 dark:text-white">{{ formatMoney(usageMap[user.id]?.total_actual_cost, usdCnyRate) }}</td>
                  <td class="px-5 py-3">
                    <div class="flex justify-end gap-1.5" @click.stop>
                      <button class="btn btn-sm btn-primary" type="button" data-test="issue-card" @click="openIssueCard(user)">
                        <Icon name="key" size="sm" />
                        开卡
                      </button>
					  <button class="btn btn-sm btn-outline" type="button" data-test="qcanvas-key-pair" @click="openQCanvasKeyPair(user)">
						QCanvas 双 Key
					  </button>
                      <button class="btn btn-sm btn-outline" type="button" data-test="row-recharge" @click="openRecharge(user)">
                        充值
                      </button>
                      <RouterLink class="btn btn-sm btn-outline" :to="`/admin/console/ai-records?user_id=${user.id}`" data-test="row-view-spend">
                        查看花费
                      </RouterLink>
                      <button class="btn btn-sm btn-outline" type="button" @click="toggleExpand(user)">
                        {{ expandedUserId === user.id ? '收起' : '查看卡片' }}
                      </button>
                    </div>
                  </td>
                </tr>
                <!-- 展开：成员的 API 卡 -->
                <tr v-if="expandedUserId === user.id">
                  <td colspan="6" class="bg-gray-50/60 px-5 py-4 dark:bg-dark-900/40">
                    <div v-if="keysLoading" class="py-4 text-center text-sm text-gray-500 dark:text-gray-400">加载卡片中…</div>
                    <template v-else>
                      <div v-if="!userKeys.length" class="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                        这名成员还没有卡。点上面的「开卡」发一张。
                      </div>
                      <table v-else class="min-w-full text-sm">
                        <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
                          <tr>
                            <th class="px-3 py-2 font-medium">卡名</th>
                            <th class="px-3 py-2 font-medium">Key</th>
                            <th class="px-3 py-2 font-medium">状态</th>
                            <th class="px-3 py-2 font-medium">额度</th>
                            <th class="px-3 py-2 font-medium">卡上花费</th>
                            <th class="px-3 py-2 font-medium">最近使用</th>
                            <th class="px-3 py-2 text-right font-medium">操作</th>
                          </tr>
                        </thead>
                        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                          <tr v-for="key in userKeys" :key="key.id">
                            <td class="px-3 py-2 font-medium text-gray-900 dark:text-white">{{ key.name || '未命名' }}</td>
                            <td class="px-3 py-2 font-mono text-xs text-gray-500 dark:text-gray-400">{{ maskKey() }}</td>
                            <td class="px-3 py-2">
                              <span class="inline-flex rounded-md px-2 py-0.5 text-xs font-medium" :class="keyStatusClass(key.status)">
                                {{ keyStatusLabel(key.status) }}
                              </span>
                            </td>
                            <td class="px-3 py-2 text-xs tabular-nums" :class="quotaWarningTextClass(quotaWarningLevel(key.quota_used, key.quota))">
                              <div>{{ key.quota > 0 ? `${formatAccountUsd(key.quota_used)} / ${formatAccountUsd(key.quota)}` : '不限额' }}</div>
                              <div
                                v-if="key.quota > 0"
                                class="mt-1 h-1.5 w-24 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600"
                              >
                                <div
                                  class="h-full rounded-full transition-all"
                                  :class="quotaWarningBarClass(quotaWarningLevel(key.quota_used, key.quota))"
                                  :style="{ width: `${Math.min(((key.quota_used || 0) / key.quota) * 100, 100)}%` }"
                                />
                              </div>
                            </td>
                            <td class="px-3 py-2 text-xs tabular-nums text-gray-900 dark:text-white">{{ formatMoney(keyUsageMap[key.id]?.total_actual_cost, usdCnyRate) }}</td>
                            <td class="px-3 py-2 text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(key.last_used_at) }}</td>
                            <td class="px-3 py-2">
                              <div class="flex justify-end gap-1.5">
                                <button
                                  class="btn btn-sm btn-outline"
                                  type="button"
                                  :disabled="keyActionBusy"
                                  @click="toggleKeyStatus(key)"
                                >
                                  {{ key.status === 'active' ? '停用' : '启用' }}
                                </button>
                                <button
                                  class="btn btn-sm btn-outline !text-red-600 dark:!text-red-400"
                                  type="button"
                                  :disabled="keyActionBusy"
                                  @click="removeKey(key)"
                                >
                                  <Icon name="trash" size="sm" />
                                </button>
                              </div>
                            </td>
                          </tr>
                        </tbody>
                      </table>
                    </template>
                  </td>
                </tr>
              </template>
              <tr v-if="!loading && !filteredUsers.length">
                <td colspan="6" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  还没有服务身份。点右上角「新增服务身份」为内部系统建档。
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- 新增服务身份：一页 3 步向导（建身份 → 开双 Key + 充值 → 明文展示一次） -->
      <div v-if="staffModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeWizard">
        <div class="w-full max-w-md rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">新增服务身份</h2>
            <span class="text-xs text-gray-400">第 {{ wizardStep }} 步 / 共 3 步</span>
          </div>

          <!-- 第 1 步：建服务身份 -->
          <template v-if="wizardStep === 1">
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              服务身份不能登录 Web，只用于持有 API 卡片和隔离调用、成本与资产归属。
            </p>
            <form class="mt-5 space-y-4" data-test="service-identity-form" @submit.prevent="createStaff">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">服务名称</label>
                <input v-model="staffForm.username" class="input" maxlength="50" placeholder="例如：QCanvas 批量出图" />
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">身份邮箱</label>
                <input v-model="staffForm.email" class="input" data-test="service-identity-email" type="email" required placeholder="qcanvas@wujie.local（仅作唯一标识）" />
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">备注（可选）</label>
                <input v-model="staffForm.notes" class="input" maxlength="200" placeholder="例如：夜间批量分镜流程" />
              </div>
              <div class="flex justify-end gap-2 pt-2">
                <button class="btn btn-outline" type="button" @click="closeWizard">取消</button>
                <button class="btn btn-primary" type="submit" data-test="wizard-step1-next" :disabled="creatingStaff">
                  {{ creatingStaff ? '创建中…' : '下一步：开卡充值' }}
                </button>
              </div>
            </form>
          </template>

          <!-- 第 2 步：开双 Key + 充值 -->
          <template v-else-if="wizardStep === 2">
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              身份「{{ staffDisplayName(wizardUser?.username, wizardUser?.email) }}」已建好。选两个不同的路由组开出双 Key，可顺手充一笔。
            </p>
            <form class="mt-5 space-y-4" data-test="wizard-pair-form" @submit.prevent="wizardIssueAndRecharge">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">视频 Key 路由组</label>
                <select v-model.number="qcanvasPairForm.videoGroupId" class="input" data-test="wizard-video-group" required>
                  <option :value="0" disabled>选择视频实际使用的组</option>
                  <option v-for="group in eligibleGroups" :key="group.id" :value="group.id" :disabled="group.id === qcanvasPairForm.mediaGroupId">
                    {{ group.name }}
                  </option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">LLM / 图片 Key 路由组</label>
                <select v-model.number="qcanvasPairForm.mediaGroupId" class="input" data-test="wizard-media-group" required>
                  <option :value="0" disabled>选择 LLM / 图片实际使用的组</option>
                  <option v-for="group in eligibleGroups" :key="group.id" :value="group.id" :disabled="group.id === qcanvasPairForm.videoGroupId">
                    {{ group.name }}
                  </option>
                </select>
              </div>
              <p v-if="eligibleGroups.length < 2 && !groupsLoading" class="text-xs text-red-600">双 Key 必须选择两个不同的可用组，当前可用组不足。</p>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">充值金额（美元，可填 0）</label>
                <input
                  v-model.number="wizardRechargeAmount"
                  class="input"
                  data-test="wizard-amount"
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="例如 50"
                />
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  按 1 美元 ≈ ¥{{ usdCnyRate }} 入账到该身份余额；不充可填 0，之后随时在行内「充值」。
                </p>
              </div>
              <div class="flex justify-end gap-2 pt-2">
                <button class="btn btn-outline" type="button" @click="closeWizard">取消</button>
                <button class="btn btn-primary" type="submit" data-test="wizard-submit" :disabled="issuing || groupsLoading || !canIssueQCanvasPair">
                  {{ issuing ? '开通中…' : '开通双 Key' }}
                </button>
              </div>
            </form>
          </template>

          <!-- 第 3 步：明文双 Key（仅这一次） -->
          <template v-else>
            <p v-if="issuedQCanvasPair?.video.key && issuedQCanvasPair?.media.key" class="mt-1 text-xs text-amber-600 dark:text-amber-300">
              两把完整 Key 只显示这一次，请立刻复制并交给 QCanvas 注册页。关掉后只能看到脱敏元数据。
            </p>
            <p v-else class="mt-1 text-xs text-amber-600 dark:text-amber-300">
              该请求已重放，明文 Key 不再返回；请不要把重试当作重新开卡。
            </p>
            <div v-if="issuedQCanvasPair?.video.key && issuedQCanvasPair?.media.key" class="mt-4 space-y-3">
              <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900">
                <div class="text-xs text-gray-500">视频 Key · {{ selectedGroupName(qcanvasPairForm.videoGroupId) }}</div>
                <div class="mt-1 break-all font-mono text-sm text-gray-900 dark:text-white" data-test="wizard-video-key">{{ issuedQCanvasPair.video.key }}</div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900">
                <div class="text-xs text-gray-500">LLM / 图片 Key · {{ selectedGroupName(qcanvasPairForm.mediaGroupId) }}</div>
                <div class="mt-1 break-all font-mono text-sm text-gray-900 dark:text-white" data-test="wizard-media-key">{{ issuedQCanvasPair.media.key }}</div>
              </div>
            </div>
            <p v-if="wizardRechargeResult" class="mt-3 text-xs text-gray-600 dark:text-gray-300" data-test="wizard-recharge-result">
              {{ wizardRechargeResult }}
            </p>
            <div class="mt-4 flex justify-end gap-2">
              <button v-if="issuedQCanvasPair?.video.key && issuedQCanvasPair?.media.key" class="btn btn-primary" type="button" data-test="wizard-copy" @click="copyIssuedQCanvasPair">
                <Icon name="copy" size="sm" />
                {{ qcanvasPairCopied ? '已复制' : '复制两把 Key' }}
              </button>
              <button class="btn btn-outline" type="button" data-test="wizard-done" @click="closeWizard">完成</button>
            </div>
          </template>
        </div>
      </div>

      <!-- 开卡弹窗 -->
      <div v-if="issueModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeIssueModal">
        <div class="w-full max-w-md rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <template v-if="!issuedKey">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              给 {{ staffDisplayName(issueTarget?.username, issueTarget?.email) }} 开卡
            </h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">这张卡强绑定该成员，卡上的每次调用都会记到这个成员头上。</p>
            <form class="mt-5 space-y-4" data-test="issue-card-form" @submit.prevent="issueCard">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">卡名</label>
                <input v-model="issueForm.name" class="input" maxlength="100" required placeholder="例如：张三-生产卡" />
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400">本地额度、频率窗口和有效期固定为不限，调用边界由上游账号现有限额决定。</p>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">计费与路由组</label>
                <select
                  v-model.number="issueForm.groupId"
                  class="input"
                  data-test="issue-card-group"
                  required
                >
                  <option :value="0" disabled>选择启用中的组</option>
                  <option v-for="group in eligibleGroups" :key="group.id" :value="group.id">
                    {{ group.name }}
                  </option>
                </select>
                <p v-if="groupsLoading" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  正在加载可用组…
                </p>
                <p v-else-if="eligibleGroups.length === 0" class="mt-1 text-xs text-red-600">
                  当前没有可绑定的启用组，暂时不能开卡。
                </p>
              </div>
              <div class="flex justify-end gap-2 pt-2">
                <button class="btn btn-outline" type="button" @click="closeIssueModal">取消</button>
                <button class="btn btn-primary" type="submit" :disabled="issuing || groupsLoading || issueForm.groupId <= 0">
                  <Icon name="key" size="sm" />
                  开卡
                </button>
              </div>
            </form>
          </template>
          <template v-else>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">开卡成功</h2>
            <p v-if="issuedKey.key" class="mt-1 text-xs text-amber-600 dark:text-amber-300">
              完整 Key 只显示这一次，请立即复制并交给成员保管。关掉这个窗口后只能看到脱敏后的元数据。
            </p>
            <p v-else class="mt-1 text-xs text-amber-600 dark:text-amber-300">
              该请求已被处理过一次（重复提交），完整 Key 已不再回显。若未保存，请删除这张卡后重新开卡。
            </p>
            <div v-if="issuedKey.key" class="mt-4 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900">
              <div class="break-all font-mono text-sm text-gray-900 dark:text-white">{{ issuedKey.key }}</div>
            </div>
            <div class="mt-4 flex justify-end gap-2">
              <button v-if="issuedKey.key" class="btn btn-primary" type="button" @click="copyIssuedKey">
                <Icon name="copy" size="sm" />
                {{ copied ? '已复制' : '复制 Key' }}
              </button>
              <button class="btn btn-outline" type="button" data-test="issue-card-done" @click="closeIssueModal">完成</button>
            </div>
          </template>
        </div>
      </div>

	  <div v-if="qcanvasPairModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeQCanvasKeyPairModal">
		<div class="w-full max-w-md rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
		  <template v-if="!issuedQCanvasPair">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">为 {{ staffDisplayName(issueTarget?.username, issueTarget?.email) }} 开通 QCanvas 双 Key</h2>
			<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">两张卡归属同一服务身份、同一余额与总账单；实际路由组由管理员明确选择。</p>
			<form class="mt-5 space-y-4" data-test="qcanvas-key-pair-form" @submit.prevent="issueQCanvasKeyPair">
			  <div>
				<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">视频 Key 路由组</label>
				<select v-model.number="qcanvasPairForm.videoGroupId" class="input" data-test="qcanvas-video-group" required>
				  <option :value="0" disabled>选择视频实际使用的组</option>
				  <option v-for="group in eligibleGroups" :key="group.id" :value="group.id" :disabled="group.id === qcanvasPairForm.mediaGroupId">{{ group.name }}</option>
				</select>
			  </div>
			  <div>
				<label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">LLM / 图片 Key 路由组</label>
				<select v-model.number="qcanvasPairForm.mediaGroupId" class="input" data-test="qcanvas-media-group" required>
				  <option :value="0" disabled>选择 LLM / 图片实际使用的组</option>
				  <option v-for="group in eligibleGroups" :key="group.id" :value="group.id" :disabled="group.id === qcanvasPairForm.videoGroupId">{{ group.name }}</option>
				</select>
			  </div>
			  <p v-if="eligibleGroups.length < 2 && !groupsLoading" class="text-xs text-red-600">双 Key 必须选择两个不同的可用组。</p>
			  <div class="flex justify-end gap-2 pt-2">
				<button class="btn btn-outline" type="button" @click="closeQCanvasKeyPairModal">取消</button>
				<button class="btn btn-primary" type="submit" :disabled="issuing || groupsLoading || !canIssueQCanvasPair">
				  <Icon name="key" size="sm" />
				  原子开通双 Key
				</button>
			  </div>
			</form>
		  </template>
		  <template v-else>
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">QCanvas 双 Key 已开通</h2>
			<p v-if="issuedQCanvasPair.video.key && issuedQCanvasPair.media.key" class="mt-1 text-xs text-amber-600 dark:text-amber-300">两把完整 Key 只显示这一次，请立刻复制并交给 QCanvas 注册页。</p>
			<p v-else class="mt-1 text-xs text-amber-600 dark:text-amber-300">该请求已重放，明文 Key 不再返回；请不要把重试当作重新开卡。</p>
			<div v-if="issuedQCanvasPair.video.key && issuedQCanvasPair.media.key" class="mt-4 space-y-3">
			  <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900"><div class="text-xs text-gray-500">视频 Key · {{ selectedGroupName(qcanvasPairForm.videoGroupId) }}</div><div class="mt-1 break-all font-mono text-sm text-gray-900 dark:text-white">{{ issuedQCanvasPair.video.key }}</div></div>
			  <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900"><div class="text-xs text-gray-500">LLM / 图片 Key · {{ selectedGroupName(qcanvasPairForm.mediaGroupId) }}</div><div class="mt-1 break-all font-mono text-sm text-gray-900 dark:text-white">{{ issuedQCanvasPair.media.key }}</div></div>
			</div>
			<div class="mt-4 flex justify-end gap-2">
			  <button v-if="issuedQCanvasPair.video.key && issuedQCanvasPair.media.key" class="btn btn-primary" type="button" @click="copyIssuedQCanvasPair"><Icon name="copy" size="sm" />{{ qcanvasPairCopied ? '已复制' : '复制两把 Key' }}</button>
			  <button class="btn btn-outline" type="button" data-test="qcanvas-key-pair-done" @click="closeQCanvasKeyPairModal">完成</button>
			</div>
		  </template>
		</div>
	  </div>

	  <!-- 行内充值弹窗 -->
	  <div v-if="rechargeModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="rechargeModalOpen = false">
		<div class="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
		  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
			给 {{ staffDisplayName(rechargeTarget?.username, rechargeTarget?.email) }} 充值
		  </h2>
		  <form class="mt-5 space-y-4" data-test="recharge-form" @submit.prevent="submitRecharge">
			<div>
			  <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">充值金额（美元）</label>
			  <input v-model.number="rechargeAmount" class="input" data-test="recharge-amount" type="number" min="0.01" step="0.01" required placeholder="例如 50" />
			  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">按 1 美元 ≈ ¥{{ usdCnyRate }} 入账，立即生效。</p>
			</div>
			<div class="flex justify-end gap-2 pt-2">
			  <button class="btn btn-outline" type="button" @click="rechargeModalOpen = false">取消</button>
			  <button class="btn btn-primary" type="submit" data-test="recharge-submit" :disabled="recharging">
				{{ recharging ? '充值中…' : '确认充值' }}
			  </button>
			</div>
		  </form>
		</div>
	  </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, AdminUser, ApiKey } from '@/types'
import type { BatchApiKeyUsageStats, BatchUserUsageStats } from '@/api/admin/dashboard'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { requestConfirmation } from '@/composables/useAppDialog'
import { DEFAULT_USD_CNY_RATE } from '@/composables/useDisplayCurrency'
import {
  CONSOLE_ERROR_ZH,
  formatAccountUsd,
  formatDateTime,
  formatMoney,
  quotaWarningBarClass,
  quotaWarningLevel,
  quotaWarningTextClass,
  staffDisplayName,
} from './consoleUtils'

const appStore = useAppStore()

const loading = ref(false)
const search = ref('')
const users = ref<AdminUser[]>([])
const usageMap = ref<Record<number, BatchUserUsageStats>>({})
// 后端 /admin/dashboard/stats 与 /admin/dashboard/users-ranking 均未提供实时汇率字段；
// 使用系统默认汇率展示，不臆造动态汇率。
const usdCnyRate = ref(DEFAULT_USD_CNY_RATE)

const filteredUsers = computed(() => {
  return users.value.filter((user) => user.role !== 'admin' && user.member_type === 'tool')
})

// ---- 成员列表 ----

async function loadStaff() {
  loading.value = true
  try {
    const res = await adminAPI.users.list(1, 100, {
      search: search.value.trim() || undefined,
      include_subscriptions: false,
      sort_by: 'created_at',
      sort_order: 'asc',
    })
    users.value = (res.items || []).filter((user) => user.role !== 'admin' && user.member_type === 'tool')
    if (users.value.length) {
      const usage = await adminAPI.dashboard.getBatchUsersUsage(users.value.map((user) => user.id))
      const map: Record<number, BatchUserUsageStats> = {}
      for (const [id, stats] of Object.entries(usage.stats || {})) {
        map[Number(id)] = stats
      }
      usageMap.value = map
    } else {
      usageMap.value = {}
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载成员列表失败', CONSOLE_ERROR_ZH))
  } finally {
    loading.value = false
  }
}

// ---- 新增成员（一页 3 步向导：建身份 → 开双 Key + 充值 → 明文展示一次） ----

const staffModalOpen = ref(false)
const creatingStaff = ref(false)
const wizardStep = ref(1)
const wizardUser = ref<AdminUser | null>(null)
const wizardRechargeAmount = ref<number>(0)
const wizardRechargeResult = ref('')
const staffForm = reactive({
  username: '',
  email: '',
  notes: '',
})

function openCreateServiceIdentity() {
  staffForm.username = ''
  staffForm.email = ''
  staffForm.notes = ''
  wizardStep.value = 1
  wizardUser.value = null
  wizardRechargeAmount.value = 0
  wizardRechargeResult.value = ''
  issuedQCanvasPair.value = null
  qcanvasPairCopied.value = false
  qcanvasPairForm.videoGroupId = 0
  qcanvasPairForm.mediaGroupId = 0
  staffModalOpen.value = true
}

function closeWizard() {
  staffModalOpen.value = false
  wizardStep.value = 1
  wizardUser.value = null
  wizardRechargeResult.value = ''
  issuedQCanvasPair.value = null
  qcanvasPairCopied.value = false
  issueTarget.value = null
}

async function createStaff() {
  creatingStaff.value = true
  try {
    const res = await adminAPI.users.create({
      email: staffForm.email.trim(),
      username: staffForm.username.trim() || undefined,
      notes: staffForm.notes.trim() || undefined,
      member_type: 'tool',
      role: 'user',
    })
    const created = res.user
    wizardUser.value = created
    issueTarget.value = created
    appStore.showSuccess('服务身份已创建，继续开卡充值')
    wizardStep.value = 2
    selectQCanvasPairDefaults()
    await loadStaff()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '创建服务身份失败', CONSOLE_ERROR_ZH))
  } finally {
    creatingStaff.value = false
  }
}

// 第 2 步提交：先开通双 Key（明文仅此一次），再按需充值；充值失败不遮挡已签发的 Key
async function wizardIssueAndRecharge() {
  if (!wizardUser.value || !canIssueQCanvasPair.value) {
    appStore.showError('请为视频与 LLM / 图片选择两个不同的可用组')
    return
  }
  issuing.value = true
  wizardRechargeResult.value = ''
  try {
    issuedQCanvasPair.value = await adminAPI.apiKeys.createQCanvasKeyPairForUser(
      wizardUser.value.id,
      { video_group_id: qcanvasPairForm.videoGroupId, media_group_id: qcanvasPairForm.mediaGroupId },
      crypto.randomUUID(),
    )
    const amount = Number(wizardRechargeAmount.value) || 0
    if (amount > 0) {
      try {
        const updated = await adminAPI.users.updateBalance(wizardUser.value.id, amount, 'add', '开卡向导充值')
        wizardRechargeResult.value = `已充值 $${amount.toFixed(2)}，当前余额 $${Number(updated.balance ?? 0).toFixed(2)}。`
      } catch (rechargeErr) {
        wizardRechargeResult.value = `充值失败：${extractApiErrorMessage(rechargeErr, '请稍后在行内「充值」重试', CONSOLE_ERROR_ZH)}。Key 已签发，请先复制。`
        appStore.showError('双 Key 已开通，但充值失败，可在列表行内重试')
      }
    }
    wizardStep.value = 3
    if (expandedUserId.value === wizardUser.value.id) await loadUserKeys(wizardUser.value.id)
    await loadStaff()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, 'QCanvas 双 Key 开通失败', CONSOLE_ERROR_ZH))
  } finally {
    issuing.value = false
  }
}

// ---- 行内充值 ----

const rechargeModalOpen = ref(false)
const rechargeTarget = ref<AdminUser | null>(null)
const rechargeAmount = ref<number>(0)
const recharging = ref(false)

function openRecharge(user: AdminUser) {
  rechargeTarget.value = user
  rechargeAmount.value = 0
  rechargeModalOpen.value = true
}

async function submitRecharge() {
  const amount = Number(rechargeAmount.value)
  if (!rechargeTarget.value || !(amount > 0)) {
    appStore.showError('请输入大于 0 的充值金额')
    return
  }
  recharging.value = true
  try {
    const updated = await adminAPI.users.updateBalance(rechargeTarget.value.id, amount, 'add', '行内充值')
    appStore.showSuccess(`已充值 $${amount.toFixed(2)}，当前余额 $${Number(updated.balance ?? 0).toFixed(2)}`)
    rechargeModalOpen.value = false
    rechargeTarget.value = null
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '充值失败', CONSOLE_ERROR_ZH))
  } finally {
    recharging.value = false
  }
}

// ---- 展开成员的卡 ----

const expandedUserId = ref<number | null>(null)
const userKeys = ref<ApiKey[]>([])
const keysLoading = ref(false)
const keyUsageMap = ref<Record<number, BatchApiKeyUsageStats>>({})

async function toggleExpand(user: AdminUser) {
  if (expandedUserId.value === user.id) {
    expandedUserId.value = null
    return
  }
  expandedUserId.value = user.id
  await loadUserKeys(user.id)
}

async function loadUserKeys(userId: number) {
  keysLoading.value = true
  try {
    const res = await adminAPI.users.getUserApiKeys(userId)
    userKeys.value = res.items || []
    if (userKeys.value.length) {
      const usage = await adminAPI.dashboard.getBatchApiKeysUsage(userKeys.value.map((key) => key.id))
      const map: Record<number, BatchApiKeyUsageStats> = {}
      for (const [id, stats] of Object.entries(usage.stats || {})) {
        map[Number(id)] = stats
      }
      keyUsageMap.value = map
    } else {
      keyUsageMap.value = {}
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载成员卡片失败', CONSOLE_ERROR_ZH))
  } finally {
    keysLoading.value = false
  }
}

// 管理员列表/详情接口约定不会回显明文或部分明文（apiKeyDTOWithoutSecret 直接清空 key），
// 但这里绝不能"失败开放"：即便后端某天意外携带了非空 key，也只显示统一占位符，
// 不对其做任何 slice/展示。完整 Key 只能来自开卡响应弹窗（issuedKey.value.key）。
function maskKey(): string {
  return '已发放·不再显示'
}

function keyStatusLabel(status: ApiKey['status']): string {
  const labels: Record<string, string> = {
    active: '在用',
    inactive: '已停用',
    disabled: '已停用',
    quota_exhausted: '额度用完',
    expired: '已过期',
  }
  return labels[status] ?? status
}

function keyStatusClass(status: ApiKey['status']): string {
  const classes: Record<string, string> = {
    active: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300',
    inactive: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300',
    disabled: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300',
    quota_exhausted: 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300',
    expired: 'bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300',
  }
  return classes[status] ?? classes.disabled
}

// ---- 卡操作 ----

const keyActionBusy = ref(false)

async function toggleKeyStatus(key: ApiKey) {
  // admin 契约只接受 active/disabled（不要发 inactive，后端会 400）
  const next = key.status === 'active' ? 'disabled' : 'active'
  if (next === 'disabled') {
    const confirmed = await requestConfirmation({
      message: `确定停用卡「${key.name || '未命名'}」？停用后该卡调用会立即失败。`,
      danger: true,
    })
    if (!confirmed) return
  }
  keyActionBusy.value = true
  try {
    await adminAPI.apiKeys.updateApiKeyFields(key.id, { status: next })
    appStore.showSuccess(next === 'active' ? '卡已启用' : '卡已停用')
    if (expandedUserId.value) await loadUserKeys(expandedUserId.value)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '切换卡状态失败', CONSOLE_ERROR_ZH))
  } finally {
    keyActionBusy.value = false
  }
}

async function removeKey(key: ApiKey) {
  const confirmed = await requestConfirmation({
    message: `确定删除卡「${key.name || '未命名'}」？删除后该卡立刻失效。`,
    danger: true,
  })
  if (!confirmed) return
  keyActionBusy.value = true
  try {
    await adminAPI.apiKeys.deleteApiKey(key.id)
    appStore.showSuccess('卡已删除')
    if (expandedUserId.value) await loadUserKeys(expandedUserId.value)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除卡失败', CONSOLE_ERROR_ZH))
  } finally {
    keyActionBusy.value = false
  }
}

// ---- 开卡 ----

const issueModalOpen = ref(false)
const issueTarget = ref<AdminUser | null>(null)
const issuing = ref(false)
const issuedKey = ref<ApiKey | null>(null)
const copied = ref(false)

type IssuedQCanvasKeyPair = { video: ApiKey; media: ApiKey }
const qcanvasPairModalOpen = ref(false)
const issuedQCanvasPair = ref<IssuedQCanvasKeyPair | null>(null)
const qcanvasPairCopied = ref(false)
const qcanvasPairForm = reactive({ videoGroupId: 0, mediaGroupId: 0 })
const activeGroups = ref<AdminGroup[]>([])
const groupsLoading = ref(true)
const issueForm = reactive({ name: '', groupId: 0 })

function canUserBindGroup(user: AdminUser, group: AdminGroup): boolean {
  if (group.status !== 'active') return false
  if (group.subscription_type === 'subscription') return false
  return !group.is_exclusive || (user.allowed_groups ?? []).includes(group.id)
}

const eligibleGroups = computed(() => {
  if (!issueTarget.value) return []
  return activeGroups.value.filter((group) => canUserBindGroup(issueTarget.value!, group))
})

function selectEligibleDefault() {
  if (!eligibleGroups.value.some((group) => group.id === issueForm.groupId)) {
    issueForm.groupId = eligibleGroups.value[0]?.id ?? 0
  }
}

function selectQCanvasPairDefaults() {
	const ids = eligibleGroups.value.map((group) => group.id)
	if (!ids.includes(qcanvasPairForm.videoGroupId)) qcanvasPairForm.videoGroupId = ids[0] ?? 0
	if (!ids.includes(qcanvasPairForm.mediaGroupId) || qcanvasPairForm.mediaGroupId === qcanvasPairForm.videoGroupId) {
		qcanvasPairForm.mediaGroupId = ids.find((id) => id !== qcanvasPairForm.videoGroupId) ?? 0
	}
}

const canIssueQCanvasPair = computed(() => {
	return eligibleGroups.value.some((group) => group.id === qcanvasPairForm.videoGroupId) &&
		eligibleGroups.value.some((group) => group.id === qcanvasPairForm.mediaGroupId) &&
		qcanvasPairForm.videoGroupId !== qcanvasPairForm.mediaGroupId
})

function selectedGroupName(groupID: number): string {
	return eligibleGroups.value.find((group) => group.id === groupID)?.name ?? `组 #${groupID}`
}

async function loadActiveGroups() {
  groupsLoading.value = true
  try {
    activeGroups.value = await adminAPI.groups.getAll()
  } catch (err) {
    activeGroups.value = []
    appStore.showError(extractApiErrorMessage(err, '加载启用组失败', CONSOLE_ERROR_ZH))
  } finally {
    groupsLoading.value = false
    if (issueModalOpen.value) selectEligibleDefault()
	// 开卡向导第 2 步与双 Key 弹窗共用 qcanvasPairForm，组加载完成后都要补默认选择
	if (qcanvasPairModalOpen.value || (staffModalOpen.value && wizardStep.value === 2)) selectQCanvasPairDefaults()
  }
}

function openIssueCard(user: AdminUser) {
  issueTarget.value = user
  issuedKey.value = null
  copied.value = false
  issueForm.name = `${staffDisplayName(user.username, user.email)}-生产卡`
  issueModalOpen.value = true
  selectEligibleDefault()
}

function openQCanvasKeyPair(user: AdminUser) {
	issueTarget.value = user
	issuedQCanvasPair.value = null
	qcanvasPairCopied.value = false
	qcanvasPairModalOpen.value = true
	selectQCanvasPairDefaults()
}

function closeIssueModal() {
  issueModalOpen.value = false
  issueTarget.value = null
  issuedKey.value = null
}

function closeQCanvasKeyPairModal() {
	qcanvasPairModalOpen.value = false
	issuedQCanvasPair.value = null
	qcanvasPairCopied.value = false
	issueTarget.value = null
}

async function issueCard() {
  if (!issueTarget.value) return
  if (!eligibleGroups.value.some((group) => group.id === issueForm.groupId)) {
    appStore.showError('请先选择计费与路由组')
    return
  }
  issuing.value = true
  try {
    // 每次开卡提交都生成新的 Idempotency-Key，避免网络重试意外重发出两张卡；
    // 若真的发生了重放，后端会返回同一张卡但 key 字段为空（见下方模板的兜底文案）。
    issuedKey.value = await adminAPI.apiKeys.createApiKeyForUser(
      issueTarget.value.id,
      {
        name: issueForm.name.trim(),
        group_id: issueForm.groupId,
        quota: 0,
      },
      crypto.randomUUID(),
    )
    appStore.showSuccess('开卡成功')
    if (expandedUserId.value === issueTarget.value.id) {
      await loadUserKeys(issueTarget.value.id)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '开卡失败', CONSOLE_ERROR_ZH))
  } finally {
    issuing.value = false
  }
}

async function issueQCanvasKeyPair() {
	if (!issueTarget.value || !canIssueQCanvasPair.value) {
		appStore.showError('请为视频与 LLM / 图片选择两个不同的可用组')
		return
	}
	issuing.value = true
	try {
		issuedQCanvasPair.value = await adminAPI.apiKeys.createQCanvasKeyPairForUser(
			issueTarget.value.id,
			{ video_group_id: qcanvasPairForm.videoGroupId, media_group_id: qcanvasPairForm.mediaGroupId },
			crypto.randomUUID(),
		)
		appStore.showSuccess('QCanvas 双 Key 开通成功')
		if (expandedUserId.value === issueTarget.value.id) await loadUserKeys(issueTarget.value.id)
	} catch (err) {
		appStore.showError(extractApiErrorMessage(err, 'QCanvas 双 Key 开通失败', CONSOLE_ERROR_ZH))
	} finally {
		issuing.value = false
	}
}

let copyResetTimeoutId: ReturnType<typeof setTimeout> | null = null

async function copyIssuedKey() {
  if (!issuedKey.value?.key) return
  try {
    await navigator.clipboard.writeText(issuedKey.value.key)
    copied.value = true
    if (copyResetTimeoutId) clearTimeout(copyResetTimeoutId)
    copyResetTimeoutId = setTimeout(() => {
      copied.value = false
      copyResetTimeoutId = null
    }, 2000)
  } catch {
    appStore.showError('复制失败，请手动选中复制')
  }
}

async function copyIssuedQCanvasPair() {
	if (!issuedQCanvasPair.value?.video.key || !issuedQCanvasPair.value.media.key) return
	try {
		await navigator.clipboard.writeText(`video=${issuedQCanvasPair.value.video.key}\nmedia=${issuedQCanvasPair.value.media.key}`)
		qcanvasPairCopied.value = true
	} catch {
		appStore.showError('复制失败，请手动复制两把 Key')
	}
}

onMounted(() => {
  void loadStaff()
  void loadActiveGroups()
})
onUnmounted(() => {
  if (copyResetTimeoutId) clearTimeout(copyResetTimeoutId)
})
</script>
