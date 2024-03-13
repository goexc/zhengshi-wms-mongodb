<script setup lang="ts">
import {nextTick, ref, reactive} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {reqAddOrUpdateMaterialCategory} from "@/api/material";
import {MaterialCategory, MaterialCategoryRequest} from "@/api/material/types.ts";
import {MaterialCategoryStatus} from "@/enums/material_category.ts";

defineOptions({
  name: "Item"
})

//获取父组件传递过来的全部路由数组
const props = defineProps(['category'])

const form = ref<MaterialCategory>(JSON.parse(JSON.stringify(props.category)))
const formRef = ref<FormInstance>()
const emit = defineEmits(['success', 'cancel'])

//更换物料分类图片
// const handleSelect = (image: string) => {
//   form.value.image = image
// }
//
// const handleRemove = (image: string) => {
//   console.log('handleRemove:', image)
// }


const rules = reactive<FormRules>({
  sort_id: [
    {
      required: true,
      message: '请设置排序值',
      trigger: ['blue', 'change'],
    },
    {
      type: 'number',
      min: 0,
      message: '排序值必须 ≥0',
      trigger: ['blue', 'change'],
    }
  ],
  name: [
    {
      required: true,
      message: '请填写物料分类名称',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  status: [
    {
      required: true,
      message: '请选择给定的状态',
      type: 'enum',
      enum: MaterialCategoryStatus,
      trigger: ['blue', 'change'],
    }
  ],
  remark: [
    {
      min: 0,
      max: 125,
      message: '备注字数 ≤ 125',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ]

})

//关闭表单
const cancel = () => {
  nextTick(() => {
    formRef.value?.clearValidate()
  })
  emit('cancel')
}
//提交表单
const submit = async () => {
  //1.表单校验
  let valid = await formRef.value?.validate((valid, fields) => {
    if (valid) {

    } else {
      console.log('fields:', fields)
    }
    return
  })

  if (!valid) {
    return
  }

  let res = await reqAddOrUpdateMaterialCategory(<MaterialCategoryRequest>{
    id: form.value.id,
    parent_id: form.value.parent_id,
    sort_id: form.value.sort_id,
    name: form.value.name,
    status: form.value.status,
    remark: form.value.remark,
  })

  if (res.code === 200) {
    await nextTick(() => {
      formRef.value?.clearValidate()
    })
    ElMessage.success(res.msg)
    emit('success')
  } else {
    ElMessage.error(res.msg)
  }
}

</script>

<template>
  <el-form
      label-width="100px"
      style="width:360px"
      :model="form"
      ref="formRef"
      :rules="rules"
  >
    <el-form-item label="物料分类名称" prop="name">
      <el-input v-model.trim="form.name" clearable/>
    </el-form-item>
    <el-form-item label="排序" prop="sort_id">
      <el-input v-model.number="form.sort_id" clearable/>
    </el-form-item>
    <el-form-item label="状态" prop="status">
      <el-select
          v-model.trim="form.status"
          clearable
        >
        <el-option v-for="(one, idx) in MaterialCategoryStatus" :key="idx" :label="one" :value="one" />
      </el-select>
    </el-form-item>
    <el-form-item label="备注" prop="remark">
      <el-input v-model.trim="form.remark" clearable/>
    </el-form-item>
    <el-form-item>
      <el-button plain @click="cancel">取消</el-button>
      <el-button type="primary" plain @click="submit">提交</el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped lang="scss">

</style>