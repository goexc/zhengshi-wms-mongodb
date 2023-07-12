import { FormRules } from "element-plus";
import { reactive } from "vue";

export const rules = reactive<FormRules>({
  parent_id: [
    {
      required: true,
      message: "必填",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
  type: [
    {
      required: true,
      message: "请选择指定的菜单类型",
      type: "enum",
      enum: [1, 2, 3],
      trigger: ["blur", "change"],
    },
  ],
  name: [
    {
      required: true,
      message: "必填",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
  icon: [
    {
      required: true,
      message: "必选",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
  path: [
    // {required: true, message: '必填', type: "string", trigger: ['blur', 'change']}
  ],
  component: [
    // {required: true, message: '必填', type: "string", trigger: ['blur', 'change']}
  ],
  sort_id: [
    {
      required: true,
      message: "必填",
      type: "number",
      trigger: ["blur", "change"],
    },
  ],
  fixed: [
    {
      required: true,
      message: "必填",
      type: "boolean",
      trigger: ["blur", "change"],
    },
  ],
  hidden: [
    {
      required: true,
      message: "必填",
      type: "boolean",
      trigger: ["blur", "change"],
    },
  ],
});
