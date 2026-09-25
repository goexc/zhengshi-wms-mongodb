import { router } from "../router";

class Scroller {
	list: Map<string, ((top: number) => void)[]> = new Map();
	private pageEvents: Map<string, (() => void)[]> = new Map();

	// 页面生命周期只在路由页触发，转发给当前页的 cl-page。
	onEvent(name: string, callback: () => void): () => void {
		const key = `${router.path()}:${name}`;
		const callbacks = this.pageEvents.get(key) ?? [];
		callbacks.push(callback);
		this.pageEvents.set(key, callbacks);
		return () => {
			const remaining = (this.pageEvents.get(key) ?? []).filter((cb) => cb != callback);
			if (remaining.length == 0) this.pageEvents.delete(key);
			else this.pageEvents.set(key, remaining);
		};
	}

	emitEvent(name: string) {
		const callbacks = this.pageEvents.get(`${router.path()}:${name}`) ?? [];
		callbacks.forEach((callback) => callback());
	}

	// 触发滚动
	emit(top: number) {
		const cbs = this.list.get(router.path()) ?? [];
		cbs.forEach((cb) => {
			cb(top);
		});
	}

	// 监听页面滚动
	on(callback: (top: number) => void) {
		const path = router.path();
		const cbs = this.list.get(path) ?? [];
		cbs.push(callback);
		this.list.set(path, cbs);
	}

	// 取消监听页面滚动
	off = (callback: (top: number) => void) => {
		const path = router.path();
		const cbs = this.list.get(path) ?? [];
		this.list.set(
			path,
			cbs.filter((cb) => cb != callback)
		);
	};
}

export const scroller = new Scroller();
