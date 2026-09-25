import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const root = process.cwd();
const errors = [];
const sourceFiles = [];

function normalize(value) {
	return value.replaceAll("\\", "/");
}

function walk(target) {
	if (!fs.existsSync(target)) return;
	const stat = fs.statSync(target);
	if (stat.isFile()) {
		if (target.endsWith(".ts") || target.endsWith(".uvue")) sourceFiles.push(target);
		return;
	}
	for (const entry of fs.readdirSync(target, { withFileTypes: true })) {
		const absolutePath = path.join(target, entry.name);
		if (entry.isDirectory()) walk(absolutePath);
		else if (entry.name.endsWith(".ts") || entry.name.endsWith(".uvue")) sourceFiles.push(absolutePath);
	}
}

for (const sourceRoot of [
	"App.uvue",
	"main.ts",
	".cool",
	"components",
	"config",
	"constants",
	"pages",
	"plugins",
	"types",
	"utils", "services", "stores", "composables"
]) {
	walk(path.join(root, sourceRoot));
}

const pagesSource = fs
	.readFileSync(path.join(root, "pages.json"), "utf8")
	.replace(/^\s*\/\/.*$/gm, "");
const pagesConfig = JSON.parse(pagesSource);
const registeredRoutes = new Set();

for (const page of pagesConfig.pages ?? []) registeredRoutes.add(`/${page.path}`);
for (const subPackage of pagesConfig.subPackages ?? []) {
	for (const page of subPackage.pages ?? []) {
		registeredRoutes.add(`/${subPackage.root}/${page.path}`);
	}
}

for (const route of registeredRoutes) {
	const pageFile = path.join(root, `${route.slice(1)}.uvue`);
	if (!fs.existsSync(pageFile)) errors.push(`pages.json: 页面文件不存在 ${route}.uvue`);
}

for (const absolutePath of sourceFiles) {
	const file = normalize(path.relative(root, absolutePath));
	const source = fs.readFileSync(absolutePath, "utf8");

	for (const match of source.matchAll(/["'`]((?:\/pages\/)[A-Za-z0-9_./-]+)(?:\?[^"'`]*)?["'`]/g)) {
		const route = match[1].replace(/\/$/, "");
		if (!registeredRoutes.has(route)) errors.push(`${file}: 引用了未注册页面 ${route}`);
	}

	for (const match of source.matchAll(/["'`](\/static\/[^"'`?#]+)["'`]/g)) {
		const asset = match[1];
		if (!fs.existsSync(path.join(root, asset.slice(1)))) {
			errors.push(`${file}: 静态资源不存在 ${asset}`);
		}
	}

	for (const match of source.matchAll(/(?:from\s+|import\s*)["']([^"']+)["']/g)) {
		const specifier = match[1];
		let target = "";
		if (specifier.startsWith("@/")) target = path.join(root, specifier.slice(2));
		else if (specifier.startsWith(".")) target = path.resolve(path.dirname(absolutePath), specifier);
		else continue;

		const candidates = [
			target,
			`${target}.ts`,
			`${target}.uvue`,
			`${target}.json`,
			path.join(target, "index.ts"),
			path.join(target, "index.uvue")
		];
		if (!candidates.some((candidate) => fs.existsSync(candidate))) {
			errors.push(`${file}: 导入目标不存在 ${specifier}`);
		}
	}
}

const pageRoot = path.join(root, "pages");
for (const absolutePath of sourceFiles) {
	const file = normalize(path.relative(root, absolutePath));
	if (!absolutePath.startsWith(pageRoot) || !file.endsWith(".uvue") || file.includes("/components/")) {
		continue;
	}
	const route = `/${file.slice(0, -".uvue".length)}`;
	if (!registeredRoutes.has(route)) errors.push(`${file}: 页面未在 pages.json 注册`);
}

if (errors.length > 0) {
	console.error([...new Set(errors)].join("\n"));
	process.exitCode = 1;
} else {
	console.log(
		`Integration validation passed (${registeredRoutes.size} routes, ${sourceFiles.length} source files).`
	);
}
