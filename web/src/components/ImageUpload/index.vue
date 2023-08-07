<script setup lang="ts">
import {reactive, ref} from "vue";
import {reqImages} from "@/api/image";
import {ImagesRequest} from "@/api/image/types.ts";
import {ElMessage} from "element-plus";
import {useUserStore} from "@/store/modules/user.ts";

defineOptions({
  name: 'Upload'
})

const props = defineProps({
  title: {
    type: String,
    default: '',
  },
  url: {// 默认图片链接
    type: String,
    default: '',
  },
  multiple: { // 是否支持多选文件
    type: Boolean,
    default: false
  },
  limit: { // 最大允许上传个数
    type: Number,
    default: 5
  }
})

const emit = defineEmits(['select', 'remove'])

const userStore = useUserStore()

const activeTab = ref<string>('upload')
const visible = ref<boolean>(false)
const loading = ref<boolean>(false)
const total = ref<number>(0)
const select = new Set()

//图片域名
const oss_domain=ref<string>(import.meta.env.VITE_OSS_DOMAIN)

// 图片提交url
const action = ref<string>(import.meta.env.VITE_IMAGE_UPLOAD)

// 点击图片上传组件
const handleCover = () => {
  visible.value = true
}

// 分页
const form = reactive<ImagesRequest>({
  page: 1,
  size: 10
})

//素材列表
const images = ref<string[]>([])


const loadImages = async () => {
  loading.value = true

  let res = await reqImages(form)
  if (res.code === 200) {
    images.value = res.data.list
    total.value = res.data.total
  } else {
    ElMessage.error(res.msg)
    images.value = []
    total.value = 0
  }
  loading.value = false
}


const handleSizeChange = () => {
  loadImages()
}
const handleCurrentChange = () => {
  loadImages()
}

const uploadHeaders = reactive({
  Authorization: userStore.token
})

// 图片文件上传成功时的钩子
const handleUpload = (res: any) => {
  console.log('图片文件上传成功时的钩子res:', res)
  if (res.code === 200) {
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
  if(select.has(image)){
    select.delete(image.url)
    emit('remove', image.url)// 让父组件删除素材链接
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
      <el-icon v-if="!url"><Plus/></el-icon>
      <el-image
          v-if="url&&url.endsWith('.svg')"
          :src="`${ oss_domain }${url}`"
      ></el-image>
      <el-image
          v-if="url&&!url.endsWith('.svg')"
          :src="`${ oss_domain }${url}_148x148`"
      ></el-image>
    </div>
    <el-dialog
        v-model="visible"
        :title="title"
        draggable
        width="800"
        :close-on-click-modal="true"
    >
      <el-tabs
          v-model="activeTab"
          type="card"
          @tab-change="handleTabChange"
      >
        <el-tab-pane label="素材库" name="images">
          <el-row :gutter="10" class="mb-2">
            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4"
                    v-for="($image, $index) in images"
                    :key="$index"
            >
              <el-image
                  v-if="$image.endsWith('.svg')"
                  class="image-item"
                  style="height: 148px;width: 148px"
                  :src="`${oss_domain}${$image}`"
                  fit="cover"
                  @click="handleSelect($image)"
              ></el-image>
              <el-image
                  v-else
                  class="image-item"
                  style="height: 148px;width: 148px"
                  :src="`${oss_domain}${$image}_148x148`"
                  fit="cover"
                  @click="handleSelect($image)"
              ></el-image>
              <div v-if="$image" class="image-selected" @click="handleSelect($image)"></div>
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
        <el-tab-pane label="上传图片" name="upload">
          <el-upload
              class="upload-demo"
              list-type="picture"
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
                一次上传不超过 5 张，每张素材不超过 3 MB
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