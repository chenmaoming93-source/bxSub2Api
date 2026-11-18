<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserOverview } from '@/api/growth'

const props = defineProps<{
  checkin: UserOverview['checkin']
}>()

const { t } = useI18n()
const guidance = computed(() => props.checkin.claimed_today ? t('growth.user.checkin.guidance') : t('growth.user.checkin.notClaimedGuidance'))
</script>

<template>
  <div class="rounded-2xl border border-teal-200/80 bg-white/80 p-4 backdrop-blur dark:border-teal-700/50 dark:bg-dark-800/80">
    <div class="flex items-center justify-between gap-2">
      <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('growth.user.checkin.title') }}</h3>
      <span class="inline-flex h-7 w-7 items-center justify-center rounded-full" :class="checkin.claimed_today ? 'bg-teal-100 text-teal-700 dark:bg-teal-500/20 dark:text-teal-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300'">
        {{ checkin.claimed_today ? '✓' : '·' }}
      </span>
    </div>
    <div class="mt-3 text-lg font-semibold" :class="checkin.claimed_today ? 'text-teal-700 dark:text-teal-300' : 'text-gray-800 dark:text-dark-100'">
      {{ checkin.claimed_today ? t('growth.user.checkin.claimed') : t('growth.user.checkin.notClaimed') }}
    </div>
    <div class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('growth.user.checkin.streak', { count: checkin.streak_days }) }}</div>
    <p class="mt-4 rounded-xl bg-teal-50 px-3 py-2 text-xs leading-5 text-teal-800 dark:bg-teal-950/40 dark:text-teal-200">{{ guidance }}</p>
  </div>
</template>
