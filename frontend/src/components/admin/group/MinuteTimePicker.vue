<template>
  <div class="mt-1 flex items-center gap-1">
    <select v-model="hour" class="input w-[3.25rem] flex-none appearance-none text-center text-sm" :aria-label="label">
      <option v-for="value in hours" :key="value" :value="value">
        {{ String(value).padStart(2, '0') }}
      </option>
    </select>
    <span class="text-sm text-gray-500">:</span>
    <select v-model="minute" class="input w-[3.25rem] flex-none appearance-none text-center text-sm" :aria-label="label" :disabled="hour === 24">
      <option v-for="value in minutes" :key="value" :value="value">
        {{ String(value).padStart(2, '0') }}
      </option>
    </select>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { parseScheduleMinute } from './modelRouteConcurrencySchedule'

const props = withDefaults(defineProps<{ modelValue: string; label?: string; allowEndOfDay?: boolean }>(), {
  label: 'Time',
  allowEndOfDay: false,
})
const emit = defineEmits<{ (event: 'update:modelValue', value: string): void }>()

const totalMinutes = computed(() => parseScheduleMinute(props.modelValue, props.allowEndOfDay) ?? 0)
const hour = computed({
  get: () => Math.floor(totalMinutes.value / 60),
  set: value => emitValue(Number(value), minute.value),
})
const minute = computed({
  get: () => totalMinutes.value % 60,
  set: value => emitValue(hour.value, Number(value)),
})
const hours = computed(() => Array.from({ length: props.allowEndOfDay ? 25 : 24 }, (_, index) => index))
const minutes = Array.from({ length: 60 }, (_, index) => index)

const emitValue = (nextHour: number, nextMinute: number) => {
  const normalizedHour = props.allowEndOfDay && nextHour === 24 ? 24 : Math.min(nextHour, 23)
  const normalizedMinute = normalizedHour === 24 ? 0 : nextMinute
  emit('update:modelValue', `${String(normalizedHour).padStart(2, '0')}:${String(normalizedMinute).padStart(2, '0')}`)
}
</script>
