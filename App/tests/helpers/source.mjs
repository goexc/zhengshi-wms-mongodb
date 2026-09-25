import { parse } from "@babel/parser";

export function stripImports(source) {
	const ast = parse(source, { sourceType: "module", plugins: ["typescript"] });
	for (const node of ast.program.body.filter((node) => node.type === "ImportDeclaration").reverse()) {
		source = source.slice(0, node.start) + source.slice(node.end);
	}
	return source;
}
