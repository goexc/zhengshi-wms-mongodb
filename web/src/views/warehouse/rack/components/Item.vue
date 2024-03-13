<script setup lang="ts">
import {nextTick, ref, reactive} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import WarehouseListItem from "@/components/Warehouse/WarehouseListItem.vue";
import {reqAddOrUpdateRack} from "@/api/warehouse_rack";
import {Rack, RackRequest} from "@/api/warehouse_rack/types.ts";
import {RackTypes} from "@/enums/rack.ts";

defineOptions({
  name: "Item"
})

//获取父组件传递过来的全部路由数组
const props = defineProps(['rack'])

const form = ref<Rack>(JSON.parse(JSON.stringify(props.rack)))
const formRef = ref<FormInstance>()
const emit = defineEmits(['success', 'cancel'])

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)


//更换货架图片
const handleSelect = (image:string) => {
  form.value.image = image
}

const handleRemove = (image:string) => {
  console.log('handleRemove:',image)
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
  type: [
    {
      required: true,
      message: '请选择货架类型',
      type: 'enum',
      enum: RackTypes,
      trigger: ['blue', 'change'],
    }
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
  image: [
    {
      required: true,
      message: '请上传货架图片',
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

  let res = await reqAddOrUpdateRack(<RackRequest>{
    id: form.value.id,
    warehouse_id: form.value.warehouse_id,
    warehouse_zone_id: form.value.warehouse_zone_id, //库区
    type: form.value.type,
    name: form.value.name,
    code: form.value.code,
    image: form.value.image,
    capacity: form.value.capacity,
    capacity_unit: form.value.capacity_unit,
    manager: form.value.manager,
    contact: form.value.contact,
    remark: form.value.remark,
  })

  if(res.code === 200){
    await nextTick(() => {
      formRef.value?.clearValidate()
    })
    ElMessage.success(res.msg)
    emit('success')
  }else{
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
    <el-form-item label="货架图片" prop="image">
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
    <el-form-item label="货架类型" prop="type">
      <el-select v-model.trim="form.type" clearable placeholder="请选择货架类型">
        <el-option v-for="(item,idx) in RackTypes" :key="idx" :label="`${idx+1}.${item}`"
                   :value="item"></el-option>
      </el-select>
    </el-form-item>
    <el-form-item label="货架名称" prop="name">
      <el-input v-model.trim="form.name" clearable/>
    </el-form-item>
    <el-form-item label="货架编号" prop="code">
      <el-input v-model.trim="form.code" clearable/>
    </el-form-item>
<!--    <el-form-item label="货架状态" prop="status">-->
<!--      <el-select v-model.trim="form.status" clearable placeholder="请选择货架状态">-->
<!--        <el-option v-for="(item,idx) in ZoneStatus" :key="idx" :label="`${idx+1}.${item}`" :value="item"></el-option>-->
<!--      </el-select>-->
<!--    </el-form-item>-->
<!--    <el-form-item label="货架地址" prop="address">
      <el-input v-model.trim="form.address" clearable/>
    </el-form-item>-->
    <el-form-item label="货架容量" prop="capacity">
      <el-input v-model.number="form.capacity" clearable/>
    </el-form-item>
    <el-form-item label="货架容量单位" prop="capacity_unit">
      <el-input v-model.trim="form.capacity_unit" clearable/>
    </el-form-item>
    <el-form-item label="负责人" prop="manager">
      <el-input v-model.trim="form.manager" clearable/>
    </el-form-item>
    <el-form-item label="联系方式" prop="contact">
      <el-input v-model.trim="form.contact" clearable/>
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