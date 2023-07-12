//时间格式
import * as dayjs from "dayjs";

export const TimeFormat = (timestamp: number) => {
  return dayjs.unix(timestamp).format("YYYY-MM-DD HH:mm:ss");
};
