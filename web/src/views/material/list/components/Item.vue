<script setup lang="ts">
import {nextTick, ref, reactive} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {Material, MaterialRequest} from "@/api/material/types.ts";
import {reqAddOrUpdateMaterial} from "@/api/material";
import MaterialCategoryListItem from "@/components/MaterialCategory/MaterialCategoryListItem.vue";

defineOptions({
  name: "Item"
})

//获取父组件传递过来的全部路由数组
const props = defineProps(['material'])

const form = ref<Material>(JSON.parse(JSON.stringify(props.material)))
const formRef = ref<FormInstance>()
const emit = defineEmits(['success', 'cancel'])

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
      message: '请填写物料型号',
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
  console.log('表单：', form.value)
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
    unit: form.value.unit,
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
    <el-form-item label="物料图片" prop="image">
      <ImageUpload
          @select="handleSelect"
          @remove="handleRemove"
          :multiple="false"
          :limit="1"
          :url="form.image"
      />
    </el-form-item>
    <MaterialCategoryListItem
      :form="form"
      />
    <el-form-item label="物料名称" prop="name">
      <el-input v-model="form.name" clearable/>
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
      <el-input v-model="form.model" clearable placeholder="用于唯一标识和区分不同种类的钢材，如：RGV4102030035"/>
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
      <el-input v-model="form.specification" clearable/>
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
      <el-input v-model.number="form.material" clearable/>
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
      <el-input v-model="form.surface_treatment" clearable/>
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
      <el-input v-model="form.strength_grade" clearable/>
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
      <el-input v-model="form.unit" clearable/>
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