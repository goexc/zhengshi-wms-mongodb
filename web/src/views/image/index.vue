<script setup lang="ts">
import {onMounted, ref} from "vue";
import {ImageItem, ImagesRequest} from "@/api/image/types.ts";
import {reqImages, reqRemoveImage} from "@/api/image";
import {ElMessage} from "element-plus";
import useClipboard from 'vue-clipboard3'
import {WarningFilled} from "@element-plus/icons-vue";
import {ImageTypes} from "@/enums/image.ts";

const { toClipboard } = useClipboard()

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

const initForm = () => {
  return <ImagesRequest>{
    page: 1,
    size: 30,
    name: '',
  }
}

const form = ref<ImagesRequest>(initForm())
const loading = ref<boolean>(false)
//素材列表
const images = ref<ImageItem[]>([])
const total = ref<number>(0)

//
const getImages = async () => {
  loading.value = true

  let res = await reqImages(form.value)
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
//
const reset = async () => {
  form.value = initForm()
  await getImages()
}

const handleSizeChange = () => {
  getImages()
}
const handleCurrentChange = () => {
  getImages()
}

//复制图片链接
const handleCopy = async (url:string)=>{
  await toClipboard(oss_domain.value+url)
  ElMessage.success('已复制图片链接：' + url)
}

//删除图片
const handleRemove = async (url:string)=>{
  let res = await reqRemoveImage(url)
  if (res.code === 200) {
    ElMessage.success('删除成功')
    await getImages()
  } else {
    ElMessage.error(res.msg)
  }
}

onMounted(async ()=>{
  await getImages()
})

</script>

<template>
<div>
  <el-form
    inline
    size="default"
  >
    <el-form-item label="名称" prop="name">
      <el-input v-model.trim="form.name" clearable placeholder="请填写图片名称"/>
    </el-form-item>
    <el-form-item label="类型">
      <el-button-group>
        <el-button plain v-for="($type, $index) in ImageTypes" :key="$index">{{$type}}</el-button>
      </el-button-group>
    </el-form-item>
    <el-form-item label=" ">
      <el-button type="primary" plain @click="getImages" icon="Search">查询</el-button>
      <el-button plain @click="reset" icon="RefreshRight">重置</el-button>
    </el-form-item>
  </el-form>
  <el-pagination
      class="m-y-2"
      v-model:current-page="form.page"
      v-model:page-size="form.size"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
      :page-sizes="[20, 30, 40, 50, 100]"
      background
      layout="total, sizes, prev, pager, next, ->,jumper"
      :pager-count="9"
      :disabled="loading"
      :hide-on-single-page="false"
      :total="total"
  ></el-pagination>
  <el-row :gutter="10" class="mb-2">
    <el-col
        class="m-y-1 align-center"
        :xs="24" :sm="24" :md="12" :lg="8" :xl="6"
        v-for="($image, $index) in images"
        :key="$index"
    >
      <el-image
          class="image-item"
          style="height: 148px;width: 296px"
          :src="`${oss_domain}${$image.url}_296x148`"
          :infinite="false"
          :hide-on-click-modal="true"
          :preview-teleported="true"
          :preview-src-list="[`${oss_domain}${$image.url}_1024x1024`]"
          fit="cover"
      ></el-image>

      <div class="image-action space-evenly color-white">
        <el-button
            type="primary"
            icon="Link"
            circle
            @click="handleCopy($image.url)"
            size="small"
        ></el-button>

        <el-popconfirm
            width="220"
            confirm-button-text="确定"
            cancel-button-text="取消"
            :icon="WarningFilled"
            icon-color="#F56C6C"
            title="确定删除改图片吗?"
            @confirm="handleRemove($image.url)"
        >
          <template #reference>
            <el-button
                type="danger"
                icon="Delete"
                circle
                size="small"
            ></el-button>
          </template>
        </el-popconfirm>
      </div>
      <div class="m-b-1">
        <el-text>
          {{$image.alt}}
        </el-text>
      </div>
    </el-col>
  </el-row>
  <el-pagination
      class="m-y-2"
      v-model:current-page="form.page"
      v-model:page-size="form.size"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
      :page-sizes="[20, 30, 40, 50, 100]"
      background
      layout="total, sizes, prev, pager, next, ->,jumper"
      :pager-count="9"
      :disabled="loading"
      :hide-on-single-page="false"
      :total="total"
  ></el-pagination>
</div>
</template>

<style scoped lang="scss">
.image-action {
  position: relative;
  bottom: 30px;
  left: 0;
  font-size: 24px;

  vertical-align: middle;
  margin: 0;
}
</style>