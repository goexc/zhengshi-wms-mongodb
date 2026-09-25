## 1.0.6（2025-06-10）
+ 【重要】新增支持鸿蒙运行环境。
+ 【重要】新增支持微信小程序运行环境。
## 1.0.5（2024-06-27）
+ 修复时区偏移量计算bug。

## 1.0.4（2024-06-02）
+ 修复 `isBefore`、`isSame`、`isAfter`、`isSameOrBefore`、`isSameOrAfter` 和 `isBetween` 等比较方法时总是以当前时间为基准比较的bug。
+ 修复安卓部分方法指定日期参数无效的bug。

## 1.0.3（2024-05-06）
+ 【重要】不再支持编译器 4.1 以下版本，请及时更新编译器为 4.1 以上版本以更好兼容后续更新迭代。
+ 【重要】适配支持 `app-js` 引擎版本。(即目前ios的js引擎版本)。
+ 【重要】`diff` api 默认单位调整为毫秒，和 [dayjs](https://day.js.org/docs/zh-CN/display/difference) 实现保持一致。
+ `diff` 方法支持传入构造的 `Dayjs` 对象，示例如下：
	
	```
	import { dayjs } from '@/uni_modules/kux-dayjs';
	
	const date1 = dayjs('2024-05-06 12:23:53');
	console.log(date1.diff(dayjs('2024-05-06'), 'M')); // 结果为743
	```
+ 修复 `startOf` 和 `endOf` 时间基准总是为当前时间的问题。
+ 调整函数签名，解决编译器4.1以上版本不兼容的问题。
+ 优化其他已知问题。
## 1.0.2（2024-03-23）
+ 支持通过字符串形式初始化实例，目前支持的字符串格式如下：
	+ YYYY-MM-DD
	+ YYYY/MM/DD
	+ YYYY-MM-DD HH
	+ YYYY-MM-DD HH:MM
	+ YYYY-MM-DD HH:MM:SS
	+ YYYY-MM-DD HH:MM:SS.millis
	+ YYYY/MM/DD HH
	+ YYYY/MM/DD HH:MM
	+ YYYY/MM/DD HH:MM:SS
	+ YYYY/MM/DD HH:MM:SS.millis
	+ ISO 8601 格式（包括 UTC 时间）
	
	示例代码如下：
	
	```
	console.log(dayjs('2023-12-13').format('YYYY-MM-DD'));
	console.log(dayjs('2024/01/01').format('YYYY-MM-DD'));
	console.log(dayjs('2023-12-13 12:23').format('YYYY-MM-DD HH:mm'));
	console.log(dayjs('2023-12-13 12:23:45').format('YYYY-MM-DD HH:mm:ss'));
	console.log(dayjs('2023-12-12 19:35:35.123').format('YYYY-MM-DD HH:mm:ss.SSS'));
	console.log(dayjs('2023-12-13T10:16:18.000Z').format('YYYY-MM-DD HH:mm:ss'));
	console.log(dayjs('2023-12-13T12:25:36.567+08:00').format('YYYY-MM-DD HH:mm:ss.SSS'));
	```
	
+ 支持通过时间戳初始化实例，由于 `uts` 联合类型限制，所以目前通过字符串形式的时间戳传入参数，示例代码如下：

	```
	console.log(dayjs(`${dayjs().valueOf()}`, true).format('YYYY-MM-DD HH:mm:ss'));
	console.log(dayjs('1683234305000', true).format('YY-MM-DD HH:mm:ss'));
	``` 
+ 补全类和函数类型签名
	> **说明**
	>
	> `HBuilderX` 版本 4.0 及以上才支持。
	
+ 修复部分场景下毫秒丢失的问题。
## 1.0.1（2024-01-30）
+ 支持web版本【hbx4.0及以上支持】
+ `解析` 增加 `YY` 年份两位数选项，示例 YY：23

## 1.0.0（2023-12-14）
初始发布
