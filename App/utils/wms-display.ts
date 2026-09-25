import { ref } from "vue";
const savedLargeText = uni.getStorageSync("wms.largeText");
export const largeText = ref(typeof savedLargeText == "boolean" ? savedLargeText as boolean : false);
export function setLargeText(value: boolean) { largeText.value = value; uni.setStorageSync("wms.largeText", value); }
