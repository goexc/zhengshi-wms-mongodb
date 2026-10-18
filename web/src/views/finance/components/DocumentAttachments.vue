<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ElMessage } from 'element-plus';
import ImageUpload from '@/components/ImageUpload/index.vue';

const props = withDefaults(defineProps<{ modelValue: string[]; scope?: string; editable?: boolean; disabled?: boolean }>(), { scope: '', editable: false, disabled: false });
const emit = defineEmits<{ 'update:modelValue': [value: string[]]; uploading: [value: boolean] }>();
// Upload completions can arrive before the parent has rendered the previous update.
const selected = ref([...props.modelValue]);
watch(() => props.modelValue, value => { selected.value = [...value]; }, { flush: 'sync' });
const fileURL = (path: string) => /^https?:\/\//i.test(path) ? path : `${import.meta.env.VITE_OSS_DOMAIN}${path.replace(/^\/+/, '')}`;
const isImage = (path: string) => /\.(png|jpe?g|gif|webp|bmp|svg)(?:\?.*)?$/i.test(path);
const label = (path: string) => path.split('/').pop() || path;
const previews = computed(() => props.modelValue.filter(isImage).map(fileURL));
function add(path: string) {
  if (props.disabled || selected.value.includes(path)) return;
  if (selected.value.length >= 10) { ElMessage.warning('最多添加 10 个附件'); return; }
  selected.value = [...selected.value, path];
  emit('update:modelValue', selected.value);
}
function remove(path: string) {
  if (props.disabled) return;
  selected.value = selected.value.filter(item => item !== path);
  emit('update:modelValue', selected.value);
}
</script>

<template>
  <div class="document-attachments">
    <ImageUpload v-if="editable" :key="scope" :title="'上传附件'" :scope="scope" file-mode :multiple="true" :limit="10" :urls="modelValue" :disabled="disabled" @select="add" @remove="remove" @uploading="emit('uploading', $event)" />
    <div v-for="path in modelValue" :key="path" class="document-file">
      <el-image v-if="isImage(path)" :src="fileURL(path)" :preview-src-list="previews" :initial-index="previews.indexOf(fileURL(path))" preview-teleported fit="contain" class="document-image" />
      <el-icon v-else :size="32"><Document /></el-icon>
      <span class="document-file-name" :title="label(path)">{{ label(path) }}</span>
      <div class="document-file-actions">
        <el-link :href="fileURL(path)" target="_blank" rel="noopener noreferrer" type="primary">查看 / 下载</el-link>
        <el-button v-if="editable" link type="danger" :disabled="disabled" @click="remove(path)">移除</el-button>
      </div>
    </div>
    <el-text v-if="!editable && modelValue.length === 0" type="info">无附件</el-text>
  </div>
</template>

<style scoped>
.document-attachments { display: flex; flex-wrap: wrap; align-items: flex-start; gap: 12px; }
.document-file { display: flex; width: 156px; min-height: 126px; flex-direction: column; align-items: center; justify-content: center; gap: 8px; border: 1px solid var(--el-border-color); border-radius: 4px; padding: 8px; }
.document-image { width: 138px; height: 90px; }
.document-file-name { width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; text-align: center; }
.document-file-actions { display: flex; align-items: center; gap: 8px; }
</style>
