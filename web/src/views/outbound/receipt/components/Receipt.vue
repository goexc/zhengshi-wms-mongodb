<script setup lang="ts">

import { reactive, ref} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import { reqReceiptOutboundOrder} from "@/api/outbound";
import {OutboundOrderReceiptRequest} from "@/api/outbound/types.ts";
import dayjs from "dayjs";

defineOptions({
  name: 'Receipt'
})

let props = defineProps(['code'])
let emit = defineEmits(['success'])

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

let rules = reactive<FormRules>({
  receipt: [
    {type: 'array', min: 1, message: '请上传收据', trigger: ['blur', 'change']},
    {type: 'array', max: 10, message: '最多上传10张收据', trigger: ['blur', 'change']},
  ],
  receipt_time: [
    {
      required: true,
      message: "必填",
      trigger: ["blur", "change"],
    },
    {
      message: '请选择签收日期',
      type: "number",
      min: 1,
      trigger: ['blur', 'change']
    },
    {
      message: '签收日期不能超过当前时间',
      type: "number",
      max: dayjs().unix(),
      trigger: ['blur', 'change']
    }
  ]
})
let formRef = ref<FormInstance>()
let form = ref<OutboundOrderReceiptRequest>({code:props.code, annex: [],receipt_time: 0})

//更换物料收据
const handleSelect = (image: string) => {
  if(!form.value.annex.includes(image)) {
    form.value.annex.push(image)
  }
}

const handleRemove = (image: string) => {
  console.log('handleRemove:', image)
  form.value.annex = form.value.annex.filter(item => item !== image)
}


//提交
let handleSubmit = async () => {
  //1.表单校验
  let valid = await formRef.value?.validate((valid, fields) => {
    if (valid) {

    } else {
      console.log('fields:', fields)
    }
    return
  })

  console.log('valid:', valid)
  if (!valid) {
    return
  }

  //提交

  let res = await reqReceiptOutboundOrder(form.value)
  if (res.code === 200) {
    emit('success')
  } else {
    ElMessage.error(res.msg)
  }
}

</script>

<template>
  <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="80px"
      size="default"
  >
    <el-form-item label="收据" prop="receipt">
      <ImageUpload
          @select="handleSelect"
          @remove="handleRemove"
          :multiple="true"
          :limit="10"
          :urls.sync="form.annex"
      />
    </el-form-item>
    <el-form-item>
        <div
            v-for="($image, $index) in form.annex"
        >
          <el-image
              v-if="$image&&$image.endsWith('.svg')"
              :key="$index"
              :src="`${ oss_domain }${$image}`"
              :infinite="true"
              :preview-teleported="true"
              :preview-src-list="[`${ oss_domain }${$image}`]"
              style="width: 148px;height: 148px;"
          ></el-image>
          <el-image
              v-if="$image&&!$image.endsWith('.svg')"
              :key="$index"
              :src="`${ oss_domain }${$image}_148x148`"
              :infinite="true"
              :preview-teleported="true"
              :preview-src-list="[`${ oss_domain }${$image}`]"
              style="width: 148px;height: 148px;"
          ></el-image>

          <div class="image-action space-evenly color-white">
            <el-button
                type="danger"
                icon="Delete"
                circle
                @click="handleRemove($image)"
                size="small"
            ></el-button>
          </div>
        </div>
    </el-form-item>
    <el-form-item label="签收时间" prop="receipt_time">
      <el-date-picker
          v-model.number="form.receipt_time"
          type="date"
          placeholder="请选择签收时间"
          size="default"
          value-format="X"
      />
    </el-form-item>
    <el-form-item>
      <el-button
          plain
          type="primary"
          size="default"
          icon="Stamp"
          @click="handleSubmit"
      >签收
      </el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped lang="scss">

.image-action {
  //background-color: rgba(204, 204, 204, 0.5);
  position: relative;
  bottom: 160px;
  left: 100px;
  font-size: 24px;

  vertical-align: middle;
  margin: 10px;

}
</style>