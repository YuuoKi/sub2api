<template>
  <ConfirmDialog
    v-if="activeAppDialog?.kind === 'confirm'"
    :show="true"
    :title="activeAppDialog.title || t('common.confirm')"
    :message="activeAppDialog.message"
    :confirm-text="activeAppDialog.confirmText"
    :cancel-text="activeAppDialog.cancelText"
    :danger="activeAppDialog.danger"
    @confirm="settleAppDialog(true)"
    @cancel="settleAppDialog(false)"
  />
  <TextPromptDialog
    v-else-if="activeAppDialog?.kind === 'prompt'"
    :show="true"
    :title="activeAppDialog.title || t('common.confirm')"
    :message="activeAppDialog.message"
    :label="activeAppDialog.label"
    :input-type="activeAppDialog.inputType"
    :placeholder="activeAppDialog.placeholder"
    :required="activeAppDialog.required"
    :confirm-text="activeAppDialog.confirmText || t('common.confirm')"
    :cancel-text="activeAppDialog.cancelText || t('common.cancel')"
    @confirm="settleAppDialog"
    @cancel="settleAppDialog(null)"
  />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ConfirmDialog from './ConfirmDialog.vue'
import TextPromptDialog from './TextPromptDialog.vue'
import { activeAppDialog, settleAppDialog } from '@/composables/useAppDialog'

const { t } = useI18n()
</script>
