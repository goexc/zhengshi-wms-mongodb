import {reactive} from "vue";
import {FormRules} from "element-plus";

export const rules = reactive<FormRules>({
  name: [
    {
      required: true,
      message: "必填",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
  code: [
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
})