//表单校验
import { FormRules } from "element-plus";
import { reactive } from "vue";

export const warehouseBinRules = reactive<FormRules>({
  name: [
    {
      required: true,
      type: "string",
      message: "请填写货位名称",
      trigger: ["blur", "change"],
    },
  ],
  code: [
    {
      type: "string",
      min: 2,
      max: 32,
      message: "货位编号长度为 2~32 个字符",
      trigger: ["blur", "change"],
    },
  ],
  capacity: [
    {
      type: "number",
      min: 0,
      message: "货位容量≥0",
      trigger: ["blur", "change"],
    },
  ],
  capacity_unit: [],
  manager: [],
  contact: [],
  remark: [],
});
