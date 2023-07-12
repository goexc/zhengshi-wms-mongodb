<script setup lang="ts">

//当前分页
import {nextTick, onMounted, reactive, ref} from "vue";
import {reqTrademarks, reqAddOrUpdateTrademark} from "@/api/product/trademark";
import {ElMessage, FormInstance, UploadProps, UploadUserFile} from "element-plus";
import useUserStore from "@/store/module/account.ts";
import {reqChangeTrademarkStatus} from "@/api/product/trademark";
import {warehouseRules} from './rules'
import {
  Warehouse,
  WarehouseRequest,
  WarehousesResponse,
  WarehouseStatusRequest
} from "@/api/product/trademark/types.ts";

onMounted(() => {
  getTrademarks()
})
const userStore = useUserStore()
const initTrademark = () => {
  return <WarehouseRequest>{
    id: '',
    type: '',
    name: '',
    code: '',
    address: '',
    capacity: 0.0,
    capacity_unit: '',
    manager: '',
    contact: '',
    image: '',
    remark: '',
  }
}

const page = ref<number>(1)
const size = ref<number>(10)
const total = ref<number>(0)
const trademarks = ref<Warehouse[]>([])
const trademark = ref<WarehouseRequest>(initTrademark())
const trademarkRef = ref<FormInstance>()



//dialog
const visible = ref<boolean>(false)
const title = ref<string>('')
//图片类型
let imagesType = ['image/jpeg', 'image/png', 'image/gif', "image/svg+xml", "image/webp"]

// 图片提交url
const action = ref(import.meta.env.VITE_IMAGE_UPLOAD)
const uploadHeaders = reactive({
  Authorization: userStore.token
})

//文件上传成功时的钩子
const handleUploadSuccess: UploadProps['onSuccess'] = (
    response
) => {
  // console.log('response:', response)
  // console.log('uploadFile:', uploadFile)
  // trademark.value.image = URL.createObjectURL(uploadFile.raw!)
  trademark.value.image = response.data.url

  //清理仓库图片的表单校验提示信息
  trademarkRef.value?.clearValidate(['image'])
}

//当超出限制时，执行的钩子函数
const handleExceed = (files: File[], uploadFiles: UploadUserFile[]) => {
  console.log('files:', files)
  console.log('fileList:', uploadFiles)
}

//封装接口：获取已有品牌分页
const getTrademarks = async () => {
  let res: WarehousesResponse = await reqTrademarks(page.value, size.value)
  if (res.code === 200) {
    total.value = res.data.total
    trademarks.value = res.data.list
  }
}

const handleSizeChange = (val: number) => {
  console.log(`${val} items per page`, size.value)
  getTrademarks()
}

const handleCurrentChange = (val: number) => {
  console.log(`page page: ${val}`, page.value)
  getTrademarks()

}

//添加
const add = () => {
  //预先清空表单校验提示信息
  nextTick(() => {
    // trademarkRef.value.resetFields()
    trademarkRef.value?.clearValidate()
  })

  title.value = '添加品牌'
  visible.value = true

}

//编辑
const edit = (row: Warehouse | WarehouseRequest) => {
  //预先清空表单校验提示信息
  nextTick(() => {
    // trademarkRef.value.resetFields()
    trademarkRef.value?.clearValidate()
  })

  title.value = '修改品牌'
  visible.value = true
  Object.assign(trademark.value, row)
  // trademark.value = row
  console.log('id:', trademark.value.id)
}
//删除
const remove = async (id: string) => {
  let req = <WarehouseStatusRequest>{id: id, status: '删除'}
  let res = await reqChangeTrademarkStatus(req)
  console.log(res)
  if(res.code === 200){
    await getTrademarks()
    ElMessage({
      type: 'success',
      message: '成功',
    })
  }else{
    ElMessage({
      type: 'error',
      message: res.msg,
    })
  }
}

const cancel = () => {
  visible.value = false
  trademark.value = initTrademark()
}

const confirm = async () => {
  //1.表单校验
  let valid = await trademarkRef.value?.validate((valid, fields) => {
    if (valid) {

    } else {
      console.log('fields:', fields)
    }
    return
  })

  if (!valid) {
    return
  }

  let res = await reqAddOrUpdateTrademark(trademark.value)
  if (res.code === 200) {
    visible.value = false
    ElMessage({
      type: 'success',
      message: trademark.value?.id.length > 0 ? '修改成功' : '添加成功'
    })

    //重新获取分页
    if (trademark.value?.id.length > 0) {
      page.value = 1
    }
    await getTrademarks()
  } else {
    ElMessage({
      type: 'error',
      message: res.msg
    })
  }
}

//
const handleBeforeUpload: UploadProps['beforeUpload'] = (rawFile) => {
  console.log(rawFile)
  if (!imagesType.includes(rawFile.type)) {
    ElMessage.error('图片类型不支持')
    return false
  } else if (rawFile.size / 1024 / 1024 > 5) {
    ElMessage.error('图片文件大小不能超过 5MB!')
    return false
  }
  return true
}

//关闭dialog
const closeDialog = () => {
  trademark.value = initTrademark()
}
</script>

<template>
  <el-card>
    <el-button type="primary" icon="Plus" @click="add">添加品牌</el-button>
    <!-- 展示已有品牌 -->
    <el-table
        class="table"
        stripe
        border
        :data="trademarks"
    >
      <el-table-column type="index" label="序号" align="center" width="120"/>
      <el-table-column prop="name" label="品牌名称" align="center" min-width="180"/>
      <el-table-column prop="logo" label="品牌 Logo" align="center"/>
      <el-table-column prop="op" label="操作" align="center">
        <template #default="{row}">
          <el-button type="primary" size="small" @click="edit(row)" icon="Edit" plain>编辑</el-button>
          <el-popconfirm
              :title="`确认删除仓库 [${row.name}] 吗？`"
              @confirm="remove(row.id)"
              width="360px"
          >
            <template #reference>
              <el-button type="danger" size="small" icon="Delete" plain>删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
        v-model:current-page="page"
        v-model:page-size="size"
        :page-sizes="[10, 20, 30, 50]"
        :background="true"
        layout="total, sizes, prev, pager, next, ->,jumper"
        :total="total"
        :pager-count="9"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
    />

    <!-- 添加、修改 -->
    <el-dialog
        v-model="visible"
        :title="title"
        @close="closeDialog"
    >
      <el-form
          class="form"
          label-width="100px"
          :model="trademark"
          ref="trademarkRef"
          :rules="warehouseRules"
      >
        <el-form-item label="仓库类型" prop="type">
          <el-input v-model="trademark.type" placeholder="请填写仓库类型"/>
        </el-form-item>
        <el-form-item label="仓库名称" prop="name">
          <el-input v-model="trademark.name" placeholder="请填写仓库名称"/>
        </el-form-item>
        <el-form-item label="仓库编号" prop="code">
          <el-input v-model="trademark.code" placeholder="请填写仓库编号"/>
        </el-form-item>
        <el-form-item label="仓库地址" prop="address">
          <el-input v-model="trademark.address" placeholder="请填写仓库地址"/>
        </el-form-item>
        <el-form-item label="仓库容量" prop="capacity">
          <el-input v-model="trademark.capacity" placeholder="请填写仓库容量"/>
        </el-form-item>
        <el-form-item label="仓库容量单位" prop="capacity_unit">
          <el-input v-model="trademark.capacity_unit" placeholder="请填写仓库容量单位，如：平方米，立方米，吨……"/>
        </el-form-item>
        <el-form-item label="负责人" prop="manager">
          <el-input v-model="trademark.manager" placeholder="请填写负责人"/>
        </el-form-item>
        <el-form-item label="联系方式" prop="contact">
          <el-input v-model="trademark.contact" placeholder="请填写联系方式"/>
        </el-form-item>
        <el-form-item label="备注" prop="remark">
          <el-input v-model="trademark.remark" placeholder="请填写备注"/>
        </el-form-item>
        <el-form-item label="品牌Logo" prop="image">
          <!-- upload组件：上传图片 -->
          <el-upload
              class="image-uploader"
              list-type="picture"
              :headers="uploadHeaders"
              :multiple="true"
              :before-upload="handleBeforeUpload"
              :on-success="handleUploadSuccess"
              drag
              name="files"
              :action="action"
              :limit="1"
              :on-exceed="handleExceed"
              :show-file-list="false"
          >
            <img v-if="trademark.image.length>0" :src="trademark.image" class="image" alt="仓库图片"/>
            <el-icon v-else class="image-uploader-icon">
              <Plus/>
            </el-icon>
          </el-upload>
        </el-form-item>
      </el-form>

      <template #footer>
      <span class="dialog-footer">
        <el-button @click="cancel">取消</el-button>
        <el-button type="primary" @click="confirm">
          确定
        </el-button>
      </span>
      </template>
    </el-dialog>
  </el-card>
</template>

<style scoped lang="scss">
.table {
  margin: 20px 0;
}

.form {
  width: 80%;
}

.image-uploader .image {
  width: 178px;
  height: 178px;
  display: block;
}
</style>

<style>
.image-uploader .el-upload {
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: var(--el-transition-duration-fast);
}

.image-uploader .el-upload:hover {
  border-color: var(--el-color-primary);
}

.el-icon.image-uploader-icon {
  font-size: 28px;
  color: #8c939d;
  width: 178px;
  height: 178px;
  text-align: center;
}
</style>