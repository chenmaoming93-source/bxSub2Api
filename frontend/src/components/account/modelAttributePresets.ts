/**
 * 模型基本属性预置清单（仅前端维护，后端不感知）。
 *
 * 作为「添加属性」下拉的快捷选项：选中预置项后自动带出属性名（key）与
 * 中文描述（description 文案通过 i18n key 引用，随界面语言切换）。
 * 用户也可以选择「自定义属性」输入任意英文属性名。
 */

export interface ModelAttributePreset {
  key: string
  descriptionKey: string
}

export const MODEL_ATTRIBUTE_PRESETS: ModelAttributePreset[] = [
  {
    key: 'context_window',
    descriptionKey: 'admin.accounts.modelAttributes.presets.contextWindow'
  },
  {
    key: 'max_output_tokens',
    descriptionKey: 'admin.accounts.modelAttributes.presets.maxOutputTokens'
  },
  {
    key: 'supports_reasoning',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsReasoning'
  },
  {
    key: 'supports_vision',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsVision'
  },
  {
    key: 'supports_audio_input',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsAudioInput'
  },
  {
    key: 'supports_pdf_input',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsPdfInput'
  },
  {
    key: 'supports_function_calling',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsFunctionCalling'
  },
  {
    key: 'supports_parallel_function_calling',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsParallelFunctionCalling'
  },
  {
    key: 'supports_tool_choice',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsToolChoice'
  },
  {
    key: 'supports_response_schema',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsResponseSchema'
  },
  {
    key: 'supports_streaming',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsStreaming'
  },
  {
    key: 'supports_prompt_caching',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsPromptCaching'
  },
  {
    key: 'supports_system_messages',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsSystemMessages'
  },
  {
    key: 'supports_web_search',
    descriptionKey: 'admin.accounts.modelAttributes.presets.supportsWebSearch'
  },
  {
    key: 'modalities',
    descriptionKey: 'admin.accounts.modelAttributes.presets.modalities'
  },
  {
    key: 'max_images_per_prompt',
    descriptionKey: 'admin.accounts.modelAttributes.presets.maxImagesPerPrompt'
  },
  {
    key: 'knowledge_cutoff',
    descriptionKey: 'admin.accounts.modelAttributes.presets.knowledgeCutoff'
  },
  {
    key: 'notes',
    descriptionKey: 'admin.accounts.modelAttributes.presets.notes'
  }
]
