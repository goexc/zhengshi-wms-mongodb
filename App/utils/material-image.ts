import { fileBaseUrl } from "@/config";

export function materialImageUrl(path: string, thumbnail: boolean = false): string {
	const value = path.trim();
	if (value.length == 0) return "";
	const url = value.startsWith("https://") || value.startsWith("http://")
		? value : fileBaseUrl + value.replace(/^\/+/, "");
	return thumbnail ? url + "_148x148" : url;
}

export function previewMaterialImage(url: string): void {
	if (url.length == 0) return;
	uni.previewImage({ urls: [url], current: 0, indicator: "default", loop: false });
}

export function saveMaterialImage(url: string): Promise<string> {
	return new Promise<string>((resolve, reject) => {
		if (url.length == 0) {
			reject(new Error("该物料尚未上传图片"));
			return;
		}
		// #ifdef H5
		previewMaterialImage(url);
		resolve("请在图片预览中长按或右键保存");
		return;
		// #endif
		// #ifndef H5
		uni.downloadFile({
			url,
			success: (download) => {
				if (download.statusCode != 200 || download.tempFilePath.length == 0) {
					reject(new Error("图片下载失败，请检查网络后重试"));
					return;
				}
				uni.saveImageToPhotosAlbum({
					filePath: download.tempFilePath,
					success: () => resolve("图片已保存到相册"),
					fail: (error) => {
						const detail = error.errMsg.toLowerCase();
						if (detail.includes("permission") || detail.includes("deny") || detail.includes("auth")) {
							reject(new Error("未获得相册保存权限，请在系统设置中允许访问相册后重试"));
						} else {
							reject(new Error("图片保存失败，请检查相册权限和剩余存储空间后重试"));
						}
					}
				});
			},
			fail: () => reject(new Error("图片下载失败，请检查网络后重试"))
		});
		// #endif
	});
}
