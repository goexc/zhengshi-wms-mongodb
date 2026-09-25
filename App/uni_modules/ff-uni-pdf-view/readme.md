# ff-uni-pdf-view

PDF 预览组件 - 支持 Android、iOS、Harmony 平台的原生 PDF 查看器。

## 平台支持

| 平台 | 支持情况 | 底层技术 |
|------|---------|---------|
| Android | ✅ | [android-pdf-viewer](https://github.com/mhiew/android-pdf-viewer) (barteksc/AndroidPdfViewer) |
| iOS | ✅ | PDFKit (Apple 官方框架，iOS 11+) |
| Harmony | ✅ | PDFKit (鸿蒙官方 PdfView 组件) |
| Web | ❌ | - |
| 小程序 | ❌ | - |

## 安装

将 `ff-uni-pdf-view` 目录放置到项目的 `uni_modules` 目录下即可。

**依赖要求：**

- HBuilderX >= 4.0
- uni-app x >= 5.0

## 使用方法

### 基础用法

```vue
<template>
  <view class="container">
    <ff-uni-pdf-view
      id="pdfView"
      :src="pdfSrc"
      :page="currentPage"
      @load="onPdfLoad"
      @error="onPdfError"
      @pageChange="onPageChange"
      style="flex: 1;"
    />
  </view>
</template>

<script setup lang="uts">
  import { createPdfViewContext } from '@/uni_modules/ff-uni-pdf-view'

  const pdfSrc = ref<string>('https://example.com/sample.pdf')
  const currentPage = ref<number>(1)
  const pageCount = ref<number>(0)
  const pdfContext = ref<IPdfViewContext | null>(null)

  // PDF 加载完成
  function onPdfLoad(detail: UTSJSONObject) {
    pageCount.value = detail['pageCount'] as number
    pdfContext.value = createPdfViewContext('pdfView')
  }

  // 加载失败
  function onPdfError(detail: UTSJSONObject) {
    console.log('PDF 加载失败:', detail['errCode'], detail['errMsg'])
  }

  // 翻页
  function onPageChange(detail: UTSJSONObject) {
    currentPage.value = detail['page'] as number
  }

  // 跳转到指定页
  function goToPage(page: number) {
    pdfContext.value?.setPage(page)
    currentPage.value = page
  }
</script>

<style>
  .container {
    flex: 1;
  }
</style>
```

### 本地文件加载

```vue
<ff-uni-pdf-view
  src="/static/document.pdf"
  :page="1"
  @load="onLoad"
/>
```

### 网络文件加载

```vue
<ff-uni-pdf-view
  src="https://example.com/report.pdf"
  :page="1"
  @load="onLoad"
  @error="onError"
/>
```

## API

### Props

| 属性 | 类型 | 必填 | 默认值 | 说明 |
|-----|------|-----|-------|------|
| src | string | ✅ | - | PDF 文件路径，支持本地路径或 http(s) 网络地址 |
| page | number | ❌ | 1 | 初始显示页码（1-based，即第一页为 1） |
| enable-page-swipe | boolean | ❌ | true | 是否允许滑动切换分页；关闭后仍可缩放并通过 `setPage` 翻页 |

### Events

| 事件名 | 参数 | 说明 |
|-------|------|------|
| load | `{ pageCount: number }` | PDF 加载完成，pageCount 为总页数 |
| error | `{ errCode: number, errMsg: string }` | 加载失败，详见错误码 |
| pageChange | `{ page: number, pageCount: number }` | 用户翻页触发，page 为当前页码 |

### 命令式 API

通过 `createPdfViewContext` 获取组件上下文对象：

```uts
import { createPdfViewContext, IPdfViewContext } from '@/uni_modules/ff-uni-pdf-view'

const pdfContext = createPdfViewContext('pdfView') as IPdfViewContext
```

**IPdfViewContext 方法：**

| 方法 | 参数 | 返回值 | 说明 |
|-----|------|-------|------|
| setPage | page: number | void | 跳转到指定页（1-based） |
| getPageCount | - | number | 获取总页数 |
| getCurrentPage | - | number | 获取当前页码（1-based） |
| reload | - | void | 重新加载当前 src |

## 错误码

| 错误码 | 说明 |
|-------|------|
| 9010001 | 文件路径无效或文件不存在 |
| 9010002 | 网络 PDF 下载失败 |
| 9010003 | PDF 文件格式错误或解析失败 |
| 9010004 | 原生组件初始化失败 |

## 平台差异说明

### Android

- **依赖库**：[android-pdf-viewer](https://github.com/mhiew/android-pdf-viewer) 3.2.0-beta.3
- **minSdkVersion**：21（Android 5.0+）
- **特性**：支持手势缩放、双击放大、平滑滚动，最大缩放倍数配置为 10.0x

### iOS

- **依赖框架**：PDFKit.framework（Apple 官方）
- **最低版本**：iOS 12+
- **特性**：零依赖，最稳定，开箱即用

### Harmony

- **依赖**：PDFKit（鸿蒙官方 `@kit.PDFKit`）
- **API**：`pdfViewManager.PdfController` + `PdfView` 组件
- **限制**：
  - `loadDocument` 不支持重复调用，二次加载前必须先 `releaseDocument`
  - 页码为 0-based（内部转换）

## 注意事项

1. **src 变化会触发重新加载**：当 `src` 属性变化时，组件会自动重新加载 PDF
2. **page 变化仅跳页**：当 `page` 属性变化时，只执行跳页操作，不会触发 `pageChange` 事件
3. **网络文件自动下载**：鸿蒙平台会先下载网络 PDF 到沙箱临时目录再加载
4. **内存管理**：组件销毁时会自动释放原生资源，避免内存泄漏

## 权限声明

- Android：需要 `android.permission.INTERNET` 权限（用于下载网络 PDF）

## 相关链接
- 插件上架到对应的插件市场
- [插件市场](https://ext.dcloud.net.cn/plugin?id=28477)

## 版本历史

### 1.0.0（2026-06-23）

- iOS：使用 PDFKit（Apple 官方框架）
- Harmony：使用官方 PdfView 组件
- Android：使用 android-pdf-viewer 第三方库
- 支持本地文件和网络 URL 加载
- 支持页码跳转和翻页事件
