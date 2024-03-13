//时间格式
import dayjs from "dayjs";

export const TimeFormat = (timestamp: number) => {
  if(timestamp<=0){
    return '-'
  }
  return dayjs.unix(timestamp).format("YYYY-MM-DD HH:mm:ss");
};

export const DateFormat = (timestamp: number) => {
  if(timestamp<=0){
    return '-'
  }
  return dayjs.unix(timestamp).format("YYYY-MM-DD");
};
