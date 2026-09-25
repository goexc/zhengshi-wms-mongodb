package uts.sdk.modules.ffUniPdfView

import android.content.Context
import com.github.barteksc.pdfviewer.PDFView
import com.github.barteksc.pdfviewer.listener.OnLoadCompleteListener
import com.github.barteksc.pdfviewer.listener.OnPageChangeListener
import com.github.barteksc.pdfviewer.listener.OnErrorListener
import io.dcloud.uts.console
import java.io.File
import java.io.FileOutputStream
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URL

/**
 * PdfViewNative
 *
 * 基于 com.github.mhiew:android-pdf-viewer:3.2.0-beta.3（barteksc 维护分支，
 * 包名仍为 com.github.barteksc.pdfviewer.PDFView）封装的 PDF 渲染原生组件。
 *
 * 供 uni-app x 的 uts 胶水层调用：uts 侧通过 element.bindAndroidView(native.pdfView)
 * 把内部的 PDFView 绑定到 native-view 中渲染。
 *
 * 注意：对外暴露的方法参数中的 Number 为 uts 的 number 映射类型（kotlin.Number），
 * 内部通过 .toInt() 转换；返回值以 kotlin Int 直接返回（可被 uts 当作 number 使用）。
 */
class PdfViewNative(
    context: Context,
    private val enablePageSwipe: Boolean
) {

    private companion object {
        // 使用 android-pdf-viewer 手势层允许的最大缩放倍率。
        const val MAX_ZOOM_SCALE = 10.0f
    }

    /** uts 胶水层会通过 element.bindAndroidView(native.pdfView) 绑定该 View */
    val pdfView: PDFView = PDFView(context, null).apply {
        setMaxZoom(MAX_ZOOM_SCALE)
    }

    /** 当前已加载（或正在加载）的 src，供 reload 复用 */
    private var currentSrc: String? = null

    /** 上一次调用 loadSrc 时传入的页码（1-based），供 reload 复用 */
    private var currentPage: Number = 1

    /** 网络下载产生的临时文件，destroy 时清理 */
    private var tmpFile: File? = null

    /** 持有 context 引用（仅用于 cacheDir，使用 applicationContext 避免泄漏 Activity） */
    private val appContext: Context = context.applicationContext

    /**
     * 加载 PDF。
     *
     * @param src 本地绝对路径或 http(s) 网络地址
     * @param page 1-based 起始页
     * @param onLoad 加载完成回调，参数为总页数（pageCount）
     * @param onError 错误回调，参数为错误码(code)与描述(msg)
     * @param onPageChange 用户滑动导致的翻页回调，page 与 pageCount 均为 1-based。
     *                     注意：setPage 触发的跳页不会回调这里。
     */
    fun loadSrc(
        src: String,
        page: Number,
        onLoad: (pageCount: Number) -> Unit,
        onError: (code: Number, msg: String) -> Unit,
        onPageChange: (page: Number, pageCount: Number) -> Unit
    ) {
        currentSrc = src
        currentPage = page

        if (src.startsWith("http://") || src.startsWith("https://")) {
            // 网络 PDF：子线程下载，完成后回主线程加载
            console.log("[PdfViewNative] loadSrc from network: $src")
            Thread {
                val downloaded = downloadToCache(src, onError)
                if (downloaded == null) {
                    return@Thread
                }
                tmpFile = downloaded
                pdfView.post {
                    loadFromFile(downloaded, page, onLoad, onError, onPageChange)
                }
            }.start()
        } else {
            // 本地路径
            var localPath = src
            if (localPath.startsWith("file://")) {
                localPath = localPath.substring("file://".length)
            }
            val file = File(localPath)
            if (!file.exists() || !file.isFile) {
                console.log("[PdfViewNative] local pdf file not exist: $localPath")
                onError(9010001, "pdf file path invalid or not exist")
                return
            }
            console.log("[PdfViewNative] loadSrc from local: $localPath")
            loadFromFile(file, page, onLoad, onError, onPageChange)
        }
    }

    /**
     * 跳转到指定页（1-based）。
     * 仅调用 jumpTo，不触发 onPageChange 回调（父组件驱动翻页不应触发回调）。
     */
    fun setPage(page: Number) {
        val target = (page.toInt() - 1).coerceAtLeast(0)
        console.log("[PdfViewNative] setPage -> index0=$target (page1=$page)")
        pdfView.jumpTo(target)
    }

    /** 返回总页数（文档加载完成前可能为 0） */
    fun getPageCount(): Number {
        return pdfView.pageCount
    }

    /** 返回当前页（1-based） */
    fun getCurrentPage(): Number {
        return pdfView.currentPage + 1
    }

    /** 用记录的 src 与 page 重新加载 */
    fun reload() {
        val src = currentSrc
        if (src == null) {
            console.log("[PdfViewNative] reload skipped: no src recorded")
            return
        }
        console.log("[PdfViewNative] reload with src=$src page=$currentPage")
        loadSrc(
            src,
            currentPage,
            onLoad = { _ -> },
            onError = { _, _ -> },
            onPageChange = { _, _ -> }
        )
    }

    /** 销毁：清理下载产生的临时文件 */
    fun destroy() {
        console.log("[PdfViewNative] destroy")
        tmpFile?.let {
            try {
                if (it.exists()) {
                    it.delete()
                    console.log("[PdfViewNative] deleted tmp file: ${it.absolutePath}")
                }
            } catch (e: Exception) {
                console.log("[PdfViewNative] delete tmp file failed: ${e.message}")
            }
        }
        tmpFile = null
    }

    // -----------------------------------------------------------------------------------------
    // 内部方法
    // -----------------------------------------------------------------------------------------

    /**
     * 把网络 PDF 下载到 cacheDir 下的临时文件。
     * 下载失败回调 onError(9010002, ...) 并返回 null。
     * 该方法在子线程调用。
     */
    private fun downloadToCache(
        src: String,
        onError: (code: Number, msg: String) -> Unit
    ): File? {
        var conn: HttpURLConnection? = null
        var input: InputStream? = null
        var output: FileOutputStream? = null
        try {
            val url = URL(src)
            conn = url.openConnection() as HttpURLConnection
            conn.connectTimeout = 30000
            conn.readTimeout = 30000
            conn.requestMethod = "GET"
            conn.instanceFollowRedirects = true
            conn.connect()
            val code = conn.responseCode
            if (code < 200 || code >= 300) {
                console.log("[PdfViewNative] network download http fail: code=$code")
                pdfView.post { onError(9010002, "download pdf from network failed") }
                return null
            }
            val tmp = File(appContext.cacheDir, "pdfview_${System.currentTimeMillis()}.pdf")
            input = conn.inputStream
            output = FileOutputStream(tmp)
            val buffer = ByteArray(8192)
            while (true) {
                val n = input.read(buffer)
                if (n <= 0) break
                output.write(buffer, 0, n)
            }
            output.flush()
            console.log("[PdfViewNative] network download ok: ${tmp.absolutePath}")
            return tmp
        } catch (e: Exception) {
            console.log("[PdfViewNative] network download exception: ${e.message}")
            pdfView.post { onError(9010002, "download pdf from network failed") }
            return null
        } finally {
            try { output?.close() } catch (_: Exception) {}
            try { input?.close() } catch (_: Exception) {}
            try { conn?.disconnect() } catch (_: Exception) {}
        }
    }

    /**
     * 用本地文件配置并加载 PDFView。该方法须在主线程调用。
     */
    private fun loadFromFile(
        file: File,
        page: Number,
        onLoad: (pageCount: Number) -> Unit,
        onError: (code: Number, msg: String) -> Unit,
        onPageChange: (page: Number, pageCount: Number) -> Unit
    ) {
        val defaultPage0 = (page.toInt() - 1).coerceAtLeast(0)
        pdfView.fromFile(file)
            .defaultPage(defaultPage0)
            // 关闭跨页滑动后，PDFView 在已缩放状态下仍允许拖拽当前页。
            .enableSwipe(enablePageSwipe)
            .swipeHorizontal(false)
            .pageSnap(enablePageSwipe)
            .autoSpacing(true)
            .pageFling(enablePageSwipe)
            .onLoad { pageCount ->
                // pageCount 为总页数
                console.log("[PdfViewNative] onLoad pageCount=$pageCount")
                onLoad(pageCount)
            }
            .onPageChange { p, pc ->
                // barteksc 回调的 p 为 0-based，pc 为总页数；对外转为 1-based
                onPageChange(p + 1, pc)
            }
            .onError { t ->
                console.log("[PdfViewNative] pdf parse error: ${t.message}")
                onError(9010003, "pdf format invalid or parse failed")
            }
            .load()
    }
}
