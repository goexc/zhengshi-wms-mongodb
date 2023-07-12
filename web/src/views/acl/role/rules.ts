//表单校验
import { FormRules } from "element-plus";
import { reactive } from "vue";

export const roleRules = reactive<FormRules>({
  name: [
    {
      required: true,
      type: "string",
      message: "请填写角色名称",
      trigger: ["blur", "change"],
    },
    {
      min: 2,
      max: 21,
      message: "角色名称长度必须是 2 ~ 21 个字符",
      trigger: ["blur", "change"],
    },
  ],
  status: [
    {
      required: true,
      type: "enum",
      enum: ["启用", "禁用", "删除"],
      message: "请选择给定的角色状态",
      trigger: ["blur", "change"],
    },
  ],
  remark: [],
});
