<template>
  <BaseDialog :show="show" :title="`SingGuard 安全检查${group ? ` · ${group.name}` : ''}`" width="wide" @close="close">
    <div class="space-y-5">
      <div class="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-dark-600">
        <div>
          <div class="font-medium text-gray-900 dark:text-white">启用请求安全检查</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">在请求转发前按分组规则调用 SingGuard。</div>
        </div>
        <button type="button" class="relative h-6 w-11 rounded-full transition-colors" :class="form.enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'" @click="form.enabled = !form.enabled">
          <span class="absolute left-1 top-1 h-4 w-4 rounded-full bg-white shadow transition-transform" :class="form.enabled ? 'translate-x-5' : 'translate-x-0'" />
        </button>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <label class="block">
          <span class="input-label">超时（毫秒）</span>
          <input v-model.number="form.timeout_ms" type="number" min="1" max="10000" class="input" />
        </label>
        <label class="block">
          <span class="input-label">异常处理</span>
          <select v-model="form.exception_action" class="input">
            <option value="allow">放行</option>
            <option value="block">阻断</option>
          </select>
        </label>
        <label class="block">
          <span class="input-label">普通事件采样率（%）</span>
          <input v-model.number="form.sample_rate" type="number" min="0" max="100" class="input" />
        </label>
      </div>

      <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
        <input v-model="form.collect_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        异步保存安全检查记录（不阻塞模型请求）
      </label>

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between">
          <div>
            <h3 class="font-medium text-gray-900 dark:text-white">风险规则</h3>
            <p class="text-xs text-gray-500 dark:text-gray-400">风险概率严格大于阈值时命中。</p>
          </div>
          <button type="button" class="btn btn-secondary" @click="addRule">添加规则</button>
        </div>
        <div v-if="form.rules.length === 0" class="rounded-lg bg-gray-50 p-4 text-sm text-gray-500 dark:bg-dark-700 dark:text-gray-400">暂无规则；启用后不会调用 SingGuard。</div>
        <div v-for="(rule, index) in form.rules" :key="index" class="mb-3 grid gap-3 rounded-lg border border-gray-200 p-3 md:grid-cols-[minmax(0,1fr)_8rem_7rem_auto] dark:border-dark-600">
          <div>
            <select v-model="rule.dimension" class="input">
              <option v-for="dimension in dimensions" :key="dimension.code" :value="dimension.code">{{ dimension.label }}</option>
            </select>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ dimensionDescription(rule.dimension) }} <span class="text-gray-400">({{ rule.dimension }})</span></p>
          </div>
          <input v-model.number="rule.threshold" type="number" min="0" max="1" step="0.01" class="input" placeholder="阈值" />
          <select v-model="rule.action" class="input">
            <option value="block">阻断</option>
            <option value="warn">告警</option>
          </select>
          <button type="button" class="btn btn-secondary" @click="removeRule(index)">删除</button>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3 pt-4">
        <button type="button" class="btn btn-secondary" @click="close">取消</button>
        <button type="button" class="btn btn-primary" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存配置' }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, SecurityCheckConfig, SecurityCheckRule } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ show: boolean; group: AdminGroup | null }>()
const emit = defineEmits<{ close: []; success: [group: AdminGroup] }>()
const appStore = useAppStore()
const saving = ref(false)
const dimensions = [
  { code: 'Dangerous_Operations_Tool_Abuse', label: '危险操作与工具滥用', description: '可能诱导系统执行破坏性或高风险工具操作。' },
  { code: 'Malicious_Code_and_Cyberattack', label: '恶意代码与网络攻击', description: '涉及恶意代码、攻击、漏洞利用或入侵行为。' },
  { code: 'Prompt_Injection_and_Jailbreak', label: '提示词注入与越狱', description: '试图绕过系统规则、修改指令优先级或突破限制。' },
  { code: 'Resource_Abuse', label: '资源滥用', description: '可能造成资源耗尽、无限循环或异常高消耗。' },
  { code: 'Sensitive_Information_Stealing', label: '敏感信息窃取', description: '试图获取密码、密钥、凭证或其他敏感信息。' },
]
const dimensionDescription = (code: string) => dimensions.find((dimension) => dimension.code === code)?.description || '未知风险维度。'
const defaultConfig = (): SecurityCheckConfig => ({
  enabled: false,
  rules: [],
  timeout_ms: 500,
  exception_action: 'allow',
  collect_enabled: false,
  sample_rate: 10,
  version: 1,
})
const form = reactive<SecurityCheckConfig>(defaultConfig())

watch(() => [props.show, props.group], () => {
  if (!props.show) return
  const config = props.group?.security_check_config || defaultConfig()
  Object.assign(form, {
    ...defaultConfig(),
    ...config,
    rules: (config.rules || []).map((rule) => ({ ...rule })),
  })
}, { immediate: true })

const addRule = () => {
  const rule: SecurityCheckRule = { dimension: dimensions[0].code, threshold: 0.8, action: 'block' }
  form.rules.push(rule)
}
const removeRule = (index: number) => form.rules.splice(index, 1)
const close = () => emit('close')
const save = async () => {
  if (!props.group) return
  if (form.timeout_ms < 1 || form.sample_rate < 0 || form.sample_rate > 100 || form.rules.some((rule) => rule.threshold < 0 || rule.threshold > 1)) {
    appStore.showError('请检查超时、采样率和规则阈值')
    return
  }
  saving.value = true
  try {
    const updated = await adminAPI.groups.updateSecurityCheck(props.group.id, {
      ...form,
      rules: form.rules.map((rule) => ({ ...rule })),
    })
    emit('success', updated)
    close()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error))
  } finally {
    saving.value = false
  }
}
</script>
