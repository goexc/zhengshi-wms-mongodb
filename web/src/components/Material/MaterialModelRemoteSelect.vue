<script setup lang="ts">
import {computed, onBeforeUnmount, ref, watch} from "vue";
import {ElMessage} from "element-plus";
import {reqMaterials} from "@/api/material";
import type {MaterialsRequest} from "@/api/material/types.ts";

defineOptions({
  name: 'MaterialModelRemoteSelect',
})

const props = defineProps<{
  modelValue: string;
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void;
}>()

const selectedModel = computed({
  get: () => props.modelValue,
  set: (value: string) => emit('update:modelValue', value ?? ''),
})

const modelOptions = ref<string[]>([])
const loading = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | undefined
let searchSequence = 0

const loadModels = async (keyword: string, sequence: number) => {
  const request: MaterialsRequest = {
    page: 1,
    size: 10,
    name: '',
    image: '',
    material: '',
    specification: '',
    model: keyword,
    surface_treatment: '',
    strength_grade: '',
  }

  try {
    const res = await reqMaterials(request)
    if (sequence !== searchSequence) {
      return
    }

    if (res.code !== 200) {
      modelOptions.value = []
      ElMessage.error(res.msg)
      return
    }

    modelOptions.value = Array.from(new Set(
      res.data.list
        .map((material) => material.model.trim())
        .filter((model) => model.length > 0),
    )).slice(0, 10)
  } catch {
    if (sequence === searchSequence) {
      modelOptions.value = []
      ElMessage.error('型号搜索失败，请稍后重试')
    }
  } finally {
    if (sequence === searchSequence) {
      loading.value = false
    }
  }
}

const remoteSearch = (query: string) => {
  const keyword = query.trim()
  const sequence = ++searchSequence

  if (searchTimer !== undefined) {
    clearTimeout(searchTimer)
    searchTimer = undefined
  }

  if (keyword !== '' && keyword !== props.modelValue) {
    emit('update:modelValue', '')
  }

  if (keyword === '') {
    loading.value = false
    modelOptions.value = []
    return
  }

  loading.value = true
  searchTimer = setTimeout(() => {
    searchTimer = undefined
    void loadModels(keyword, sequence)
  }, 300)
}

watch(() => props.modelValue, (value) => {
  if (value === '') {
    modelOptions.value = []
  }
})

onBeforeUnmount(() => {
  searchSequence++
  if (searchTimer !== undefined) {
    clearTimeout(searchTimer)
  }
})
</script>

<template>
  <el-select
      v-model="selectedModel"
      clearable
      filterable
      remote
      remote-show-suffix
      autocomplete="off"
      :loading="loading"
      :remote-method="remoteSearch"
      :reserve-keyword="false"
      placeholder="请输入并选择型号"
      loading-text="正在搜索型号"
      no-match-text="未找到匹配型号"
      no-data-text="请输入型号关键字"
  >
    <el-option
        v-for="model in modelOptions"
        :key="model"
        :label="model"
        :value="model"
    />
  </el-select>
</template>
