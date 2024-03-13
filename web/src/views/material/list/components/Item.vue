<script setup lang="ts">
import {nextTick, ref, reactive} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import { MaterialRequest} from "@/api/material/types.ts";
import {reqAddOrUpdateMaterial} from "@/api/material";
import MaterialCategoryListItem from "@/components/MaterialCategory/MaterialCategoryListItem.vue";

defineOptions({
  name: "Item"
})

//获取父组件传递过来的全部路由数组
const props = defineProps(['material'])

const form = ref<MaterialRequest>({
  id: props.material.id,
  category_id: props.material.category_id,
  name: props.material.name,
  image: props.material.image,
  material: props.material.material,
  specification: props.material.specification,
  model: props.material.model,
  surface_treatment: props.material.surface_treatment,
  strength_grade: props.material.strength_grade,
  quantity: props.material.quantity,
  unit: props.material.unit,
  remark: props.material.remark,
  price: props.material.price,
})
const formRef = ref<FormInstance>()
const emit = defineEmits(['success', 'cancel'])

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

//更换物料图片
const handleSelect = (image: string) => {
  form.value.image = image
}

const handleRemove = (image: string) => {
  console.log('handleRemove:', image)
}

const rules = reactive<FormRules>({
  category_id: [
    {
      required: true,
      message: '请选择物料分类',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  name: [
    {
      required: true,
      message: '请填写物料名称',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  model: [
    {
      required: true,
      message: '请填写物料规格',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  material: [
    {
      required: false,
      message: '请填写物料材质，如：碳钢、不锈钢、合金钢等',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  specification: [
    {
      required: true,
      message: '请填写物料规格：包括长度、宽度、厚度等尺寸信息',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  surface_treatment: [
    {
      required: false,
      message: '请填写物料表面处理方式，如：热镀锌、喷涂等。',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  strength_grade: [
    {
      required: false,
      message: '请填写物料强度等级，如：Q235、Q345',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  quantity: [
    {
      required: false,
      message: '安全库存必须≥0',
      type: 'number',
      min: 0,
      trigger: ['blue', 'change'],
    }
  ],
  unit: [
    {
      required: false,
      message: '请填写物料计量单位，如：个、箱、千克等',
      type: 'string',
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
  ],
  price: [
    {
      min: 0,
      message: '单价必须≥0',
      type: 'number',
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

  let res = await reqAddOrUpdateMaterial(<MaterialRequest>{
    id: form.value.id,
    category_id: form.value.category_id,
    name: form.value.name,
    model: form.value.model,
    image: form.value.image,
    material: form.value.material,
    specification: form.value.specification,
    surface_treatment: form.value.surface_treatment,
    strength_grade: form.value.strength_grade,
    quantity: form.value.quantity,
    unit: form.value.unit,
    remark: form.value.remark,
    price: form.value.price,
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
    <el-form-item label="物料图片" prop="image">
      <ImageUpload
          @select="handleSelect"
          @remove="handleRemove"
          :multiple="false"
          :limit="1"
          :url="form.image"
      />
    </el-form-item>
    <el-form-item label="">
      <el-image
          v-if="form.image&&form.image.endsWith('.svg')"
          :src="`${ oss_domain }${form.image}`"
          :infinite="true"
          :preview-teleported="true"
          :preview-src-list="[`${ oss_domain }${form.image}`]"
          style="width: 148px;height: 148px;"
      ></el-image>
      <el-image
          v-if="form.image&&!form.image.endsWith('.svg')"
          :src="`${ oss_domain }${form.image}_148x148`"
          :infinite="true"
          :preview-teleported="true"
          :preview-src-list="[`${ oss_domain }${form.image}`]"
          style="width: 148px;height: 148px;"
      ></el-image>
    </el-form-item>
    <MaterialCategoryListItem
      :form="form"
      />
    <el-form-item label="物料名称" prop="name">
      <el-input v-model.trim="form.name" clearable/>
    </el-form-item>
    <el-form-item label="" prop="model">
      <template #label>
        <el-popover placement="left" width="420">
          <template #reference>
            <el-text size="default">
              型号 <el-icon><Warning/></el-icon>
            </el-text>
          </template>
          用于唯一标识和区分不同种类的钢材，如：RGV4102030035
        </el-popover>
      </template>
      <el-input v-model.trim="form.model" clearable placeholder="用于唯一标识和区分不同种类的钢材，如：RGV4102030035"/>
    </el-form-item>
    <el-form-item label="" prop="specification">
      <template #label>
        <el-popover placement="left" width="400">
          <template #reference>
            <el-text size="default">
              规格 <el-icon><Warning/></el-icon>
            </el-text>
          </template>
          物料规格：包括长度、宽度、厚度等尺寸信息
        </el-popover>
      </template>
      <el-input v-model.trim="form.specification" clearable/>
    </el-form-item>
    <el-form-item label="" prop="material">
      <template #label>
        <el-popover placement="left" width="400">
          <template #reference>
            <el-text size="default">
              材质 <el-icon><Warning/></el-icon>
            </el-text>
          </template>
          物料材质，如：碳钢、不锈钢、合金钢等
        </el-popover>
      </template>
      <el-input v-model.trim="form.material" clearable/>
    </el-form-item>
    <el-form-item label="" prop="surface_treatment">
      <template #label>
        <el-popover placement="left" width="400">
          <template #reference>
            <el-text size="default">
              表面处理 <el-icon><Warning/></el-icon>
            </el-text>
          </template>
          表面处理方式，如：热镀锌、喷涂等。
        </el-popover>
      </template>
      <el-input v-model.trim="form.surface_treatment" clearable/>
    </el-form-item>
    <el-form-item label="" prop="strength_grade">
      <template #label>
        <el-popover placement="left" width="400">
          <template #reference>
            <el-text size="default">
              强度等级 <el-icon><Warning/></el-icon>
            </el-text>
          </template>
          物料强度等级，如：Q235、Q345
        </el-popover>
      </template>
      <el-input v-model.trim="form.strength_grade" clearable/>
    </el-form-item>
    <el-form-item label="安全库存" prop="quantity">
      <el-input-number
          v-model.trim="form.quantity"
          :controls="false"
          :precision="3"
          :value-on-clear="1"
          clearable/>
    </el-form-item>
    <el-form-item label="" prop="unit">
      <template #label>
        <el-popover placement="left" width="400">
          <template #reference>
            <el-text size="default">
              计量单位 <el-icon><Warning/></el-icon>
            </el-text>
          </template>
          物料计量单位，如：个、箱、千克等
        </el-popover>
      </template>
      <el-input v-model.trim="form.unit" clearable/>
    </el-form-item>
    <el-form-item label="备注" prop="remark">
      <el-input v-model.trim="form.remark" clearable/>
    </el-form-item>
    <el-form-item label="单价" prop="price">
      <el-input-number
          v-model="form.price"
          :controls="false"
          :min="0"
          clearable/>
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