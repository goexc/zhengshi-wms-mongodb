<script setup lang="ts">
import {computed, onBeforeUnmount, reactive, ref} from "vue";
import {reqImages} from "@/api/image";
import {ImageItem, ImagesRequest} from "@/api/image/types.ts";
import {ElMessage} from "element-plus";
import type {UploadFile, UploadInstance, UploadRawFile} from 'element-plus';
import {useUserStore} from "@/store/modules/user.ts";

defineOptions({
  name: 'Upload'
})

const props = defineProps({
  title: {
    type: String,
    default: '',
  },
  urls: {// 默认图片链接
    type: Array,
    default: [],
  },
  url: {// 默认图片链接
    type: String,
    default: '',
  },
  multiple: { // 是否支持多选文件
    type: Boolean,
    default: false
  },
  scope: { type: String, default: '' },
  fileMode: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  limit: { // 最大允许上传个数
    type: Number,
    default: 5
  }
})

const emit = defineEmits(['select', 'remove', 'uploading'])

const userStore = useUserStore()

const activeTab = ref<string>('upload')
const visible = ref<boolean>(false)
const loading = ref<boolean>(false)
const total = ref<number>(0)
const select = new Set()
const uploader = ref<UploadInstance>()
const uploads = reactive(new Set<number>())
let disposed = false
let imageGeneration = 0
const chosen = computed(() => props.urls.filter((item): item is string => typeof item === 'string'))
const uploadData = computed(() => props.scope ? {scope: props.scope} : {})
const fileAccept = '.png,.jpg,.jpeg,.gif,.webp,.bmp,.pdf,.doc,.docx,.xls,.xlsx,.ofd'
const isImage = (path: string) => /\.(png|jpe?g|gif|webp|bmp|svg)(\?.*)?$/i.test(path)
const fileURL = (path: string) => /^https?:\/\//i.test(path) ? path : `${oss_domain.value}${path.replace(/^\/+/, '')}`
function beforeUpload(file: UploadRawFile) {
  if (disposed || props.disabled) return false
  if (props.fileMode) {
  if (chosen.value.length + uploads.size >= props.limit) { ElMessage.warning(`最多添加 ${props.limit} 个附件`); return false }
  if (!/\.(png|jpe?g|gif|webp|bmp|pdf|docx?|xlsx?|ofd)$/i.test(file.name)) { ElMessage.error('请选择图片、PDF、OFD、Word 或 Excel 文件'); return false }
  if (file.size > 20 * 1024 * 1024) { ElMessage.error('单个附件不能超过 20 MB'); return false }
  }
  uploads.add(file.uid)
  emit('uploading', true)
  return true
}
function finishUpload(file?: UploadFile) {
  if (file) uploads.delete(file.uid)
  emit('uploading', uploads.size > 0)
}
function uploadError(_error: Error, file: UploadFile) {
  if (disposed || !uploads.has(file.uid)) return
  finishUpload(file)
  ElMessage.error('附件上传失败，请重试')
}
function removeUpload(file: UploadFile) {
  finishUpload(file)
  if (!props.fileMode) return
  const response = file.response as { code?: number; data?: { url?: string } } | undefined
  if (response?.code === 200 && response.data?.url) emit('remove', response.data.url)
}
onBeforeUnmount(() => { disposed = true; ++imageGeneration; uploader.value?.abort(); uploads.clear(); emit('uploading', false) })

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

// 图片提交url
const action = ref<string>(import.meta.env.VITE_IMAGE_UPLOAD)

// 点击图片上传组件
const handleCover = () => {
  if (props.disabled) return
  if (props.fileMode && uploads.size === 0) uploader.value?.clearFiles()
  visible.value = true
}

// 分页
const form = ref<ImagesRequest>({
  page: 1,
  size: 10,
  name: ''
})

//素材列表
const images = ref<ImageItem[]>([])


const loadImages = async () => {
  const generation = ++imageGeneration
  loading.value = true
  try {
  const res = await reqImages({...form.value, ...(props.scope ? {scope: props.scope} : {})})
  if (generation !== imageGeneration) return
  if (res.code !== 200) throw new Error(res.msg)
    images.value = res.data.list
    total.value = res.data.total
  } catch (error) {
    if (generation !== imageGeneration) return
    ElMessage.error(error instanceof Error ? error.message : '素材加载失败，请重试')
    images.value = []
    total.value = 0
  } finally { if (generation === imageGeneration) loading.value = false }
}


const handleSizeChange = () => {
  loadImages()
}
const handleCurrentChange = () => {
  loadImages()
}

const initForm = () => {
  return <ImagesRequest>{
    page: 1,
    size: 30,
    name: '',
  }
}
//
const reset = async () => {
  form.value = initForm()
  await loadImages()
}

const uploadHeaders = computed(() => ({
  Authorization: userStore.token
}))

// 图片文件上传成功时的钩子
const handleUpload = (res: any, file?: UploadFile) => {
  if (disposed || (file && !uploads.has(file.uid))) return
  finishUpload(file)
  if (res.code === 200 && typeof res.data?.url === 'string' && res.data.url) {
    ElMessage.success(res.msg)
    emit('select', res.data.url)
  } else {
    ElMessage.error(res.msg)
  }
}


// 文件超出个数限制时的钩子
const handleExceed = (files: FileList, fileList: FileList) => {
  // files: 本次选择上传的文件列表
  // fileList: 已经上传成功的文件列表
  console.log('files:', files)
  console.log('fileList:', fileList)
  ElMessage.warning(`当前限制选择 ${props.limit} 个文件，本次选择了 ${files.length} 个文件，共选择了 ${files.length + fileList.length} 个文件`)
}

// 选择封面素材
const handleSelect = (image: string) => {
  if (disposed || props.disabled) return
  if (props.fileMode) {
    if (chosen.value.includes(image)) { emit('remove', image); return }
    if (chosen.value.length + uploads.size >= props.limit) { ElMessage.warning(`最多添加 ${props.limit} 个附件`); return }
    emit('select', image)
    return
  }
  if (select.has(image)) {
    // select.delete(image.url)
    // emit('remove', image.url)// 让父组件删除素材链接
    select.delete(image)
    emit('remove', image)// 让父组件删除素材链接
    return;
  }

  if (select.size < props.limit) {
    // 向父组件传递素材链接
    emit('select', image)
    return
  }

  if (select.size >= props.limit) {
    ElMessage.warning(`最多选择 ${props.limit} 张素材`)
    console.log('图片上限', select.size)
    return
  }
}

const handleTabChange = async () => {
  if (activeTab.value === 'images') {
    await loadImages()
  }
}


</script>

<template>
  <div class="upload-cover">
    <div
        class="el-upload el-upload--picture-card"
        @click="handleCover"
    >
      <!--
            <el-icon v-if="!url"><Plus/></el-icon>
            <el-image
                v-if="url&&url.endsWith('.svg')"
                :src="`${ oss_domain }${url}`"
            ></el-image>
            <el-image
                v-if="url&&!url.endsWith('.svg')"
                :src="`${ oss_domain }${url}_148x148`"
            ></el-image>
            -->
      <el-icon>
        <Plus/>
      </el-icon>
    </div>
    <el-dialog
        v-model.trim="visible"
        :title="title"
        draggable
        width="1000"
        :close-on-click-modal="!fileMode || uploads.size === 0"
        :close-on-press-escape="!fileMode || uploads.size === 0"
        :show-close="!fileMode || uploads.size === 0"
    >
      <el-tabs
          v-model.trim="activeTab"
          type="card"
          @tab-change="handleTabChange"
      >
        <el-tab-pane label="素材库" name="images">
          <el-form
              inline
              size="small"
              class="m-b-2"
              @submit.prevent="loadImages"
          >
            <el-form-item label="名称" prop="name">
              <el-input v-model.trim="form.name" clearable placeholder="请填写图片名称"/>
            </el-form-item>
            <el-form-item label=" ">
              <el-button type="primary" plain native-type="submit" icon="Search">查询</el-button>
              <el-button plain @click="reset" icon="RefreshRight">重置</el-button>
            </el-form-item>
          </el-form>
          <el-row :gutter="10" class="mb-2">
            <el-col :xs="24" :sm="24" :md="12" :lg="8" :xl="6"
                    v-for="($image, $index) in images"
                    :key="$index"
            >
              <el-image
                  v-if="isImage($image.url)"
                  class="image-item"
                  style="height: 148px;width: 148px"
                  :src="fileMode ? fileURL($image.url) : `${oss_domain}${$image.url}_148x148`"
                  fit="cover"
                  @click="handleSelect($image.url)"
              ></el-image>
              <el-button v-else style="height: 148px; width: 148px; white-space: normal" icon="Document" @click="handleSelect($image.url)">{{ $image.alt || $image.url.split('/').pop() }}</el-button>
              <div class="m-b-1">
                <el-text size="small">
                  {{$image.alt}}
                </el-text>
              </div>
              <el-tag v-if="fileMode && chosen.includes($image.url)" type="success">已选择</el-tag>
              <div v-if="!fileMode && $image" class="image-selected" @click="handleSelect($image.url)"></div>
            </el-col>
          </el-row>
          <el-pagination
              v-model:current-page="form.page"
              v-model:page-size="form.size"
              @size-change="handleSizeChange"
              @current-change="handleCurrentChange"
              :page-sizes="[10, 20, 30, 40]"
              background
              layout="total, sizes, prev, pager, next, ->,jumper"
              :pager-count="9"
              :disabled="loading"
              :hide-on-single-page="false"
              :total="total"
          ></el-pagination>
        </el-tab-pane>
        <el-tab-pane :label="fileMode ? '上传附件' : '上传图片'" name="upload">
          <el-upload
              ref="uploader"
              class="upload-demo"
              :list-type="fileMode ? 'text' : 'picture'"
              :data="uploadData"
              :accept="fileMode ? fileAccept : undefined"
              :before-upload="beforeUpload"
              :on-error="uploadError"
              :on-remove="removeUpload"
              :disabled="disabled"
              :headers="uploadHeaders"
              :on-success="handleUpload"
              drag
              name="files"
              :action="action"
              :multiple="multiple"
              :limit="limit"
              :on-exceed="handleExceed"
          >
            <i class="el-icon-upload"></i>
            <div class="el-upload__text">将文件拖到此处，或<em>点击上传</em></div>
            <template #tip>
              <div class="el-upload__tip">
                {{ fileMode ? `最多 ${limit} 个附件，单个不超过 20 MB；支持图片、PDF、OFD、Word、Excel` : '一次上传不超过 5 张，每张素材不超过 3 MB' }}
              </div>
            </template>
          </el-upload>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
</style>
