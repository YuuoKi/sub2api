<template>
  <span
    class="inline-flex max-w-full shrink-0 items-center gap-1 truncate text-xs font-normal text-ui-text-muted"
    data-testid="proxy-ad-banner"
  >
    <span class="truncate">{{ t('admin.proxies.ad.inline') }}</span>
  </span>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
