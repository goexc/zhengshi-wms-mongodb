//表单校验
import { FormRules } from "element-plus";
import { reactive } from "vue";

export const warehouseRules = reactive<FormRules>({
  type: [
    {
      required: true,
      type: "string",
      message: "请选择仓库类型",
      trigger: ["blur", "change"],
    },
    {
      type: "enum",
      enum: import.meta.env.VITE_WAREHOUSE_TYPES,
      message: "请选择给定的仓库类型",
      trigger: ["blur", "change"],
    },
  ],
  name: [
    {
      required: true,
      type: "string",
      message: "请填写仓库名称",
      trigger: ["blur", "change"],
    },
  ],
  code: [
    {
      type: "string",
      min: 2,
      max: 32,
      message: "仓库编号长度为 2~32 个字符",
      trigger: ["blur", "change"],
    },
  ],
  address: [],
  capacity: [
    {
      type: "float",
      min: 0,
      message: "请填写仓库容量",
      trigger: ["blur", "change"],
    },
  ],
  capacity_unit: [],
  manager: [],
  contact: [],
  remark: [],
  image: [
    {
      required: true,
      type: "string",
      message: "请上传仓库图片",
      trigger: ["blur", "change"],
    },
  ],
});
