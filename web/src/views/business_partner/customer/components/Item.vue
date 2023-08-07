<script setup lang="ts">
import {nextTick, ref, reactive, onMounted} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {reqAddOrUpdateCustomer} from "@/api/customer";
import {Customer, CustomerRequest} from "@/api/customer/types.ts";
import {CustomerTypes} from "@/enums/customer.ts";

defineOptions({
  name: "Item"
})

//获取父组件传递过来的全部路由数组
const props = defineProps(['customer'])

const form = ref<Customer>(JSON.parse(JSON.stringify(props.customer)))
const formRef = ref<FormInstance>()
const emit = defineEmits(['success', 'cancel'])

//更换客户图片
const handleSelect = (image: string) => {
  form.value.image = image
}

const handleRemove = (image: string) => {
  console.log('handleRemove:', image)
}


const rules = reactive<FormRules>({
  name: [
    {
      required: true,
      message: '请填写客户名称',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  code: [
    {
      required: true,
      message: '请上传客户编号',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  legal_representative: [
    {
      required: true,
      message: '请填写法定代表人',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  unified_social_credit_identifier: [
    {
      required: true,
      message: '请填写统一社会信用代码',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  manager: [
    {
      required: true,
      message: '请填写负责人',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  contact: [
    {
      required: true,
      message: '请填写联系方式',
      type: 'string',
      trigger: ['blue', 'change'],
    }
  ],
  email: [
    {
      required: false,
      message: 'Email格式错误',
      type: 'email',
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

  let res = await reqAddOrUpdateCustomer(<CustomerRequest>{
    id: form.value.id,
    type: form.value.type,
    name: form.value.name,
    code: form.value.code,
    image: form.value.image,
    legal_representative: form.value.legal_representative,
    unified_social_credit_identifier: form.value.unified_social_credit_identifier,
    manager: form.value.manager,
    contact: form.value.contact,
    email: form.value.email,
    address: form.value.address,
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
      label-width="130px"
      style="width:360px"
      :model="form"
      ref="formRef"
      :rules="rules"
  >
    <el-form-item label="客户图片" prop="image">
      <ImageUpload
          @select="handleSelect"
          @remove="handleRemove"
          :multiple="false"
          :limit="1"
          :url="form.image"
      />
    </el-form-item>
    <el-form-item label="客户类型" prop="name">
      <el-select
          v-model="form.type"
          placeholder="请选择客户类型"
          clearable
      >
        <el-option v-for="(one, idx) in CustomerTypes" :key="idx" :label="one" :value="one"/>
      </el-select>
    </el-form-item>
    <el-form-item label="客户名称" prop="name">
      <el-input v-model="form.name" clearable/>
    </el-form-item>
    <el-form-item label="客户编号" prop="code">
      <el-input v-model="form.code" clearable/>
    </el-form-item>
    <el-form-item label="法定代表人" prop="legal_representative">
      <el-input v-model.number="form.legal_representative" clearable/>
    </el-form-item>
    <el-form-item v-if="form.type === '个人'" label="身份证号码" prop="unified_social_credit_identifier">
      <el-input v-model="form.unified_social_credit_identifier" clearable/>
    </el-form-item>
    <el-form-item v-if="form.type !== '个人'" label="统一社会信用代码" prop="unified_social_credit_identifier">
      <el-input v-model="form.unified_social_credit_identifier" clearable/>
    </el-form-item>
    <el-form-item label="负责人" prop="manager">
      <el-input v-model="form.manager" clearable/>
    </el-form-item>
    <el-form-item label="联系方式" prop="contact">
      <el-input v-model="form.contact" clearable/>
    </el-form-item>
    <el-form-item label="Email" prop="email">
      <el-input v-model="form.email" clearable/>
    </el-form-item>
    <el-form-item label="地址" prop="address">
      <el-input v-model="form.address" clearable/>
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