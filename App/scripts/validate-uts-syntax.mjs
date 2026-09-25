import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { parse as parseScript } from "@babel/parser";
import { parse as parseSfc } from "@vue/compiler-sfc";

const root = process.cwd();
const roots = [
	"App.uvue",
	"main.ts",
	".cool",
	"components",
	"config",
	"constants",
	"data",
	"locales",
	"pages",
	"plugins",
	"types",
	"utils", "services", "stores", "composables"
];
const errors = [];
let checked = 0;
const forbiddenWarehouseRoutes = [
	"/pages/inventory/task/",
	"/pages/inventory/container/",
];

function visit(node, file, lineOffset = 0) {
	if (node == null || typeof node !== "object") return;

	if (node.type === "TSArrayType" && node.elementType?.type === "TSTypeLiteral") {
		errors.push(`${file}:${(node.loc?.start.line ?? 1) + lineOffset} UTS 不支持直接声明对象字面量数组类型，请提取为具名 interface`);
	}

	if (node.type === "ImportDeclaration" && node.source?.value === "@dcloudio/uni-app") {
		for (const specifier of node.specifiers ?? []) {
			if (specifier.imported?.name === "onLoad") {
				errors.push(`${file}:${(specifier.loc?.start.line ?? 1) + lineOffset} uni-app x 页面应使用全局 onLoad，不要从 @dcloudio/uni-app 导入`);
			}
		}
	}

	if (node.type === "CallExpression" && node.callee?.type === "Identifier" && node.callee.name === "onLoad") {
		const callback = node.arguments?.[0];
		const parameter = callback?.params?.[0];
		if (parameter != null && parameter.typeAnnotation == null) {
			errors.push(`${file}:${(node.loc?.start.line ?? 1) + lineOffset} onLoad 查询参数必须显式声明为 UTSJSONObject`);
		}
	}

	if (node.type === "BinaryExpression" && containsDirectCharCodeAt(node)) {
		errors.push(`${file}:${(node.loc?.start.line ?? 1) + lineOffset} charCodeAt 在 UTS 中返回 Number?，参与运算前必须使用 ?? 0 消除空值`);
	}

	for (const [key, value] of Object.entries(node)) {
		if (key === "loc" || key === "start" || key === "end") continue;
		if (Array.isArray(value)) {
			for (const item of value) visit(item, file, lineOffset);
		} else {
			visit(value, file, lineOffset);
		}
	}
}

function containsDirectCharCodeAt(node) {
	if (node == null || typeof node !== "object") return false;
	if (
		node.type === "CallExpression" &&
		node.callee?.type === "MemberExpression" &&
		node.callee.property?.type === "Identifier" &&
		node.callee.property.name === "charCodeAt"
	) {
		return true;
	}
	if (node.type === "LogicalExpression" && node.operator === "??") return false;
	return containsDirectCharCodeAt(node.left) || containsDirectCharCodeAt(node.right);
}

function parseTypeScript(source, file, lineOffset = 0) {
	try {
		const ast = parseScript(source, {
			sourceType: "module",
			plugins: ["typescript", "topLevelAwait"],
			errorRecovery: true,
		});
		for (const error of ast.errors ?? []) {
			const conditionalDuplicate = error.message.includes("already been declared") && source.includes("#if");
			if (!conditionalDuplicate) {
				const line = (error.loc?.line ?? 1) + lineOffset;
				errors.push(`${file}:${line} ${error.message}`);
			}
		}
		visit(ast, file, lineOffset);
	} catch (error) {
		const line = (error.loc?.line ?? 1) + lineOffset;
		errors.push(`${file}:${line} ${error.message}`);
	}
}

function validateFile(absolutePath) {
	const file = path.relative(root, absolutePath).replaceAll("\\", "/");
	const source = fs.readFileSync(absolutePath, "utf8");
	checked += 1;
	for (const route of forbiddenWarehouseRoutes) {
		if (source.includes(route)) {
			errors.push(`${file}: 仍引用已退役仓储页面 ${route}`);
		}
	}
	if (file.endsWith(".ts")) {
		parseTypeScript(source, file);
		return;
	}

	const { descriptor, errors: sfcErrors } = parseSfc(source, { filename: file });
	for (const error of sfcErrors) errors.push(`${file}: ${String(error)}`);
	for (const block of [descriptor.script, descriptor.scriptSetup]) {
		if (block != null) parseTypeScript(block.content, file, block.loc.start.line - 1);
	}
}

function walk(target) {
	if (!fs.existsSync(target)) return;
	const stat = fs.statSync(target);
	if (stat.isFile()) {
		if ((target.endsWith(".uvue") || target.endsWith(".ts")) && !target.endsWith(".d.ts")) validateFile(target);
		return;
	}
	for (const entry of fs.readdirSync(target, { withFileTypes: true })) {
		const absolutePath = path.join(target, entry.name);
		if (entry.isDirectory()) walk(absolutePath);
		else if ((entry.name.endsWith(".uvue") || entry.name.endsWith(".ts")) && !entry.name.endsWith(".d.ts")) validateFile(absolutePath);
	}
}

for (const directory of roots) walk(path.join(root, directory));

const pagesSource = fs.readFileSync(path.join(root, "pages.json"), "utf8");
for (const route of forbiddenWarehouseRoutes) {
	if (pagesSource.includes(route.replace(/^\//, ""))) {
		errors.push(`pages.json: 仍注册已退役仓储页面 ${route}`);
	}
}

if (errors.length > 0) {
	console.error(errors.join("\n"));
	process.exitCode = 1;
} else {
	console.log(`UTS static validation passed (${checked} files).`);
}
