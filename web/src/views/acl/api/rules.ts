import { FormRules } from "element-plus";
import { reactive } from "vue";

export const rules = reactive<FormRules>({
  type: [
    {
      required: true,
      message: "请选择指定的Api类型",
      type: "enum",
      enum: [1, 2],
      trigger: ["blur", "change"],
    },
  ],
  required: [
    {
      required: true,
      message: "必填",
      type: "boolean",
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
  sort_id: [
    {
      required: true,
      message: "必填",
      type: "number",
      trigger: ["blur", "change"],
    },
  ],
  method: [
    {
      required: true,
      message: "必填",
      type: 'enum',
      enum: ['', 'GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD'],
      trigger: ["blur", "change"],
    }
  ],
  uri: [
    {
      required: true,
      message: "必填",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
});
