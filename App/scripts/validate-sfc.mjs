import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { compileScript, compileTemplate, parse } from "@vue/compiler-sfc";

const root = process.cwd();
const sourceRoots = ["App.uvue", "components", "pages"];
const errors = [];
let checked = 0;

function formatError(file, error) {
	const message = error instanceof Error ? error.message : String(error);
	return `${file}: ${message}`;
}

function validateFile(absolutePath) {
	const file = path.relative(root, absolutePath).replaceAll("\\", "/");
	const source = fs.readFileSync(absolutePath, "utf8");
	const id = `sfc-${checked}`;
	checked += 1;

	try {
		const result = parse(source, { filename: file });
		for (const error of result.errors) errors.push(formatError(file, error));
		if (result.errors.length > 0) return;

		const descriptor = result.descriptor;
		if (descriptor.script != null || descriptor.scriptSetup != null) {
			compileScript(descriptor, { id });
		}
		if (descriptor.template != null) {
			const templateResult = compileTemplate({
				id,
				filename: file,
				source: descriptor.template.content,
				compilerOptions: {
					expressionPlugins: ["typescript"]
				}
			});
			for (const error of templateResult.errors) errors.push(formatError(file, error));
		}
	} catch (error) {
		errors.push(formatError(file, error));
	}
}

function walk(target) {
	if (!fs.existsSync(target)) return;
	const stat = fs.statSync(target);
	if (stat.isFile()) {
		if (target.endsWith(".uvue")) validateFile(target);
		return;
	}

	for (const entry of fs.readdirSync(target, { withFileTypes: true })) {
		const absolutePath = path.join(target, entry.name);
		if (entry.isDirectory()) walk(absolutePath);
		else if (entry.name.endsWith(".uvue")) validateFile(absolutePath);
	}
}

for (const sourceRoot of sourceRoots) walk(path.join(root, sourceRoot));

if (errors.length > 0) {
	console.error(errors.join("\n"));
	process.exitCode = 1;
} else {
	console.log(`SFC validation passed (${checked} files).`);
}
