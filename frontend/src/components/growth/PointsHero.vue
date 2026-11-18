<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UserOverview } from '@/api/growth'
import PointsAmount from './PointsAmount.vue'
import CheckinCard from './CheckinCard.vue'

defineProps<{
  overview: UserOverview
}>()

const emit = defineEmits<{
  aiwork: []
}>()

const { t } = useI18n()
</script>

<template>
  <section class="relative overflow-hidden rounded-2xl border border-teal-100 bg-gradient-to-br from-teal-50 via-white to-sky-50 p-5 shadow-sm dark:border-teal-900/40 dark:from-teal-950/50 dark:via-dark-800 dark:to-sky-950/30 sm:p-7">
    <div class="pointer-events-none absolute -right-10 -top-12 h-44 w-44 rounded-full bg-teal-200/30 blur-3xl dark:bg-teal-500/10" />
    <div class="relative grid gap-6 lg:grid-cols-[1fr_20rem] lg:items-stretch">
      <div class="flex flex-col justify-center">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p class="text-sm font-medium text-teal-700 dark:text-teal-300">{{ t('growth.user.hero.available') }}</p>
            <div class="mt-1 flex items-baseline gap-2">
              <span class="text-4xl font-bold tracking-tight tabular-nums text-gray-950 dark:text-white sm:text-5xl">{{ overview.available_points.toLocaleString() }}</span>
              <span class="text-sm text-gray-500 dark:text-dark-300">{{ t('growth.common.points') }}</span>
            </div>
            <p v-if="overview.available_points === 0" class="mt-2 text-sm text-gray-500 dark:text-dark-300">{{ t('growth.user.hero.emptyHint') }}</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm bg-white/80 dark:bg-dark-700/80" @click="emit('aiwork')">{{ t('growth.user.hero.goAiwork') }}</button>
        </div>
        <div class="mt-5 flex flex-wrap gap-x-6 gap-y-2 text-sm text-gray-600 dark:text-dark-300">
          <span>{{ t('growth.user.hero.totalEarned') }} <b class="text-teal-700 dark:text-teal-300">+{{ overview.total_earned.toLocaleString() }}</b></span>
          <span>{{ t('growth.user.hero.totalSpent') }} <b class="text-rose-600 dark:text-rose-300">−{{ overview.total_spent.toLocaleString() }}</b></span>
          <span>{{ t('growth.user.hero.monthEarned') }} <PointsAmount :value="overview.month_earned" direction="IN" signed /></span>
        </div>
      </div>
      <CheckinCard v-if="overview.checkin" :checkin="overview.checkin" />
    </div>
  </section>
</template>
