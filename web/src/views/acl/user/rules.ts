//表单校验
import { FormRules } from "element-plus";
import { reactive } from "vue";

export const userRules = reactive<FormRules>({
  name: [
    {
      required: true,
      type: "string",
      message: "请填写账号名称",
      trigger: ["blur", "change"],
    },
    {
      min: 2,
      max: 21,
      message: "账号名称长度必须是 2 ~ 21 个字符",
      trigger: ["blur", "change"],
    },
  ],
  password: [
    {
      required: true,
      type: "string",
      message: "请填写密码",
      trigger: ["blur", "change"],
    },
    {
      min: 6,
      max: 16,
      message: "密码长度应为 6 ~ 16 个字符",
      trigger: ["blur", "change"],
    },
  ],
  sex: [
    {
      required: true,
      type: "enum",
      enum: ["男", "女"],
      message: "请选择性别",
      trigger: ["blur", "change"],
    },
  ],
  roles_id: [
    {
      required: true,
      type: "array",
      message: "请选择角色",
      trigger: ["blur", "change"],
    },
  ],
  department_id: [
    {
      required: true,
      type: "string",
      message: "请选择部门",
      trigger: ["blur", "change"],
    },
  ],
  mobile: [
    {
      required: true,
      type: "string",
      message: "请填写手机号码",
      trigger: ["blur", "change"],
    },
    {
      len: 11,
      message: "手机号码格式：18810509066",
      trigger: ["blur", "change"],
    },
  ],
  email: [
    {
      required: true,
      type: "email",
      message: "请填写 Email",
      trigger: ["blur", "change"],
    },
  ],
  status: [
    {
      required: true,
      type: "enum",
      enum: ["启用", "禁用", "删除"],
      message: "请选择给定的用户状态",
      trigger: ["blur", "change"],
    },
  ],
  remark: [],
});
