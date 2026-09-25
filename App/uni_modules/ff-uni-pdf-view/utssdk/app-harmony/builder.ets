/**
 * uni-pdf-view 鸿蒙平台 - PdfView 声明式UI构建函数
 * 用于 native-view 绑定，创建 PDF 渲染组件
 *
 * 属性取值依据（官方 PdfView 组件文档）：
 * - controller: pdfViewManager.PdfController（必填，加载/翻页由其驱动）
 * - pageFit: pdfService.PageFit.FIT_WIDTH（页面适配宽度）
 * - isContinuous: 是否允许连续滚动到相邻页
 * - showScroll: 是否显示滚动条
 *
 * ArkTS 严格模式合规说明：
 * - params 使用 ESObject 类型，访问属性用括号访问 + as 类型断言
 * - @Builder 函数无返回类型声明（ArkUI 装饰器函数特性）
 */

import { PdfView, pdfViewManager, pdfService } from '@kit.PDFKit'

@Builder
export function buildPdfView(params: ESObject) {
  PdfView({
    controller: params['controller'] as pdfViewManager.PdfController,
    pageFit: pdfService.PageFit.FIT_WIDTH,
    isContinuous: params['enablePageSwipe'] as boolean,
    showScroll: params['enablePageSwipe'] as boolean
  })
    .id('uni_pdf_view_harmony')
    .width('100%')
    .height('100%')
}
