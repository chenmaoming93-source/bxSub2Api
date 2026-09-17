<template>
  <div class="mt-2">
    <div v-if="code === 'api_key_id'" class="relative">
      <input
        v-model="apiKeySearchText"
        class="input"
        required
        placeholder="搜索 API Key 名称、Key 或用户"
        @input="onAPIKeySearchInput"
      />
      <div v-if="apiKeySearchResults.length" class="absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <button
          v-for="key in apiKeySearchResults"
          :key="key.id"
          type="button"
          class="flex w-full flex-col gap-0.5 px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
          @click="selectAPIKey(key)"
        >
          <span class="flex items-center gap-2">
            <span class="truncate font-medium text-gray-800 dark:text-gray-100">{{ key.name || `#${key.id}` }}</span>
            <span class="shrink-0 text-xs text-gray-400">#{{ key.id }}</span>
          </span>
          <span class="flex items-center gap-2">
            <span class="truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ key.masked_key }}</span>
            <span v-if="key.user_email || key.user_name" class="shrink-0 truncate text-xs text-gray-400">{{ key.user_name || key.user_email }}</span>
          </span>
        </button>
      </div>
    </div>
    <Select
      v-else-if="searchableOptions"
      :model-value="modelValue"
      :options="options"
      searchable
      clearable
      :placeholder="placeholder"
      :search-placeholder="`搜索${displayName}`"
      @update:model-value="updateValue"
    />
    <input
      v-else
      :value="modelValue"
      class="input"
      required
      :type="valueType === 'int64' ? 'number' : 'text'"
      :min="valueType === 'int64' ? 1 : undefined"
      :placeholder="placeholder"
      @input="updateFromInput"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import Select from '@/components/common/Select.vue'
import { searchApiKeys, type SimpleApiKey } from '@/api/admin/usage'
import type { DimensionCode } from '@/api/admin/dynamicTokenStatistics'
import type { SelectOption } from '@/types'

const props = withDefaults(defineProps<{
  modelValue?: string | number
  code: DimensionCode
  displayName: string
  valueType: 'int64' | 'string'
  options?: SelectOption[]
  useSelector?: boolean
  selectedLabel?: string
}>(), {
  modelValue: '',
  options: () => [],
  useSelector: false,
  selectedLabel: ''
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: string | number): void
  (event: 'change', value: string | number): void
  (event: 'selected-label', label: string): void
}>()

const searchableOptions = computed(() => props.useSelector)
const placeholder = computed(() => searchableOptions.value ? `请选择${props.displayName}` : '具体值（不支持任意值）')
const apiKeySearchText = ref(props.selectedLabel)
const apiKeySearchResults = ref<SimpleApiKey[]>([])
let apiKeySearchTimer: ReturnType<typeof setTimeout> | undefined
let apiKeySearchController: AbortController | undefined
let apiKeySearchVersion = 0

watch(() => props.selectedLabel, value => {
  if (props.code === 'api_key_id' && value !== apiKeySearchText.value) apiKeySearchText.value = value
})
watch(() => props.modelValue, value => {
  if (props.code === 'api_key_id' && (value === '' || value === undefined)) apiKeySearchText.value = ''
})

function updateValue(value: string | number | boolean | null) {
  const normalized = typeof value === 'number' || typeof value === 'string' ? value : ''
  emit('update:modelValue', normalized)
  emit('change', normalized)
}
function updateFromInput(event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update:modelValue', value)
  emit('change', value)
}
function onAPIKeySearchInput() {
  emit('update:modelValue', '')
  emit('change', '')
  emit('selected-label', '')
  apiKeySearchResults.value = []
  if (apiKeySearchTimer) clearTimeout(apiKeySearchTimer)
  apiKeySearchController?.abort()
  const keyword = apiKeySearchText.value.trim()
  if (keyword.length < 2) return
  const version = ++apiKeySearchVersion
  apiKeySearchTimer = setTimeout(async () => {
    const controller = new AbortController()
    apiKeySearchController = controller
    try {
      const results = await searchApiKeys(undefined, keyword, { signal: controller.signal })
      if (version === apiKeySearchVersion) apiKeySearchResults.value = results
    } catch {
      if (version === apiKeySearchVersion && !controller.signal.aborted) apiKeySearchResults.value = []
    }
  }, 400)
}
function selectAPIKey(key: SimpleApiKey) {
  const user = key.user_name || key.user_email
  const label = user ? `${key.name} · ${key.masked_key} · ${user}` : `${key.name} · ${key.masked_key}`
  apiKeySearchText.value = label
  apiKeySearchResults.value = []
  emit('update:modelValue', key.id)
  emit('change', key.id)
  emit('selected-label', label)
}

onBeforeUnmount(() => {
  if (apiKeySearchTimer) clearTimeout(apiKeySearchTimer)
  apiKeySearchController?.abort()
})
</script>
