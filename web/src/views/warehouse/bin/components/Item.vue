<script setup lang="ts">
import {nextTick, ref, reactive, onMounted} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {reqAddOrUpdateWarehouseBin} from "@/api/warehouse_bin";
import {WarehouseBin, WarehouseBinRequest} from "@/api/warehouse_bin/types.ts";

defineOptions({
  name: "Item"
})

//获取父组件传递过来的全部路由数组
const props = defineProps(['bin'])

const form = ref<WarehouseBin>(JSON.parse(JSON.stringify(props.bin)))
const formRef = ref<FormInstance>()
const emit = defineEmits(['success', 'cancel'])

//更换货架图片
const handleSelect = (image: string) => {
  form.value.image = image
}

const handleRemove = (image: string) => {
  console.log('handleRemove:', image)
}


const rules = reactive<FormRules>({
  warehouse_id: [
    {
      required: true,
      message: "请选择仓库",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
  warehouse_zone_id: [
    {
      required: true,
      message: "请选择库区",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
  warehouse_rack_id: [
    {
      required: true,
      message: "请选择货架",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
  name: [
    {
      required: true,
      message: '请填写货架名称',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  code: [
    {
      required: true,
      message: '请上传货架编号',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  capacity: [
    {
      required: true,
      message: '请填写货架容量',
      type: 'number',
      trigger: ['blue', 'change'],
    },
    {
      min: 0,
      message: '货架容量必须 > 0',
      type: 'number',
      trigger: ['blue', 'change'],
    }
  ],
  capacity_unit: [
    {
      required: true,
      message: '请填写货架容量单位',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  // manager: [
  //   {
  //     required: false,
  //     message: '请填写负责人',
  //     type: 'string',
  //     trigger: ['blue', 'change'],
  //   }
  // ],
  // contact: [
  //   {
  //     required: false,
  //     message: '请填写联系方式',
  //     type: 'string',
  //     trigger: ['blue', 'change'],
  //   }
  // ],
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

  let res = await reqAddOrUpdateWarehouseBin(<WarehouseBinRequest>{
    id: form.value.id,
    warehouse_id: form.value.warehouse_id,
    warehouse_zone_id: form.value.warehouse_zone_id, //库区
    warehouse_rack_id: form.value.warehouse_rack_id, //货架
    name: form.value.name,
    code: form.value.code,
    image: form.value.image,
    capacity: form.value.capacity,
    capacity_unit: form.value.capacity_unit,
    manager: form.value.manager,
    contact: form.value.contact,
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

    <WarehouseListItem
        :form="form"
    />
    <ZoneListItem
        :form="form"
    />
    <RackListItem
        :form="form"
    />
    <el-form-item label="货架图片" prop="image">
      <ImageUpload
          @select="handleSelect"
          @remove="handleRemove"
          :multiple="false"
          :limit="1"
          :url="form.image"
      />
    </el-form-item>
    <el-form-item label="货架名称" prop="name">
      <el-input v-model="form.name" clearable/>
    </el-form-item>
    <el-form-item label="货架编号" prop="code">
      <el-input v-model="form.code" clearable/>
    </el-form-item>
    <el-form-item label="货架容量" prop="capacity">
      <el-input v-model.number="form.capacity" clearable/>
    </el-form-item>
    <el-form-item label="货架容量单位" prop="capacity_unit">
      <el-input v-model="form.capacity_unit" clearable/>
    </el-form-item>
    <el-form-item label="负责人" prop="manager">
      <el-input v-model="form.manager" clearable/>
    </el-form-item>
    <el-form-item label="联系方式" prop="contact">
      <el-input v-model="form.contact" clearable/>
    </el-form-item>
    <el-form-item label="备注" prop="remark">
      <el-input v-model="form.remark" clearable/>
    </el-form-item>
    <el-form-item>
      <el-button plain @click="cancel">取消</el-button>
      <el-button type="primary" plain @click="submit">提交</el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped lang="scss">
.el-image {
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: var(--el-transition-duration-fast);
}

.el-image:hover {
  border-color: var(--el-color-primary);
}

.image-slot {
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 28px;
  color: #8c939d;
  width: 178px;
  height: 178px;
  text-align: center;
}
</style>