import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";

test("HBuilderX wrapper fails on hidden compiler diagnostics and restores environment", { skip: process.platform !== "win32" }, () => {
	const tempParent = path.resolve(os.tmpdir());
	const fixture = fs.mkdtempSync(path.join(tempParent, "wms-build-script-test-"));
	const app = path.join(fixture, "App");
	const hx = path.join(fixture, "HBuilderX fixture");
	const nodePath = path.join(hx, "plugins", "node", "node.exe");
	const cliPath = path.join(hx, "plugins", "uniapp-cli-vite", "node_modules", "@dcloudio", "vite-plugin-uni", "bin", "uni.js");
	try {
		fs.mkdirSync(path.join(app, "scripts"), { recursive: true });
		fs.mkdirSync(path.dirname(nodePath), { recursive: true });
		fs.mkdirSync(path.dirname(cliPath), { recursive: true });
		fs.copyFileSync(process.execPath, nodePath);
		fs.copyFileSync(new URL("../scripts/build-hbuilderx.ps1", import.meta.url), path.join(app, "scripts", "build-hbuilderx.ps1"));
		fs.writeFileSync(path.join(app, "package.json"), "{}");
		fs.writeFileSync(path.join(app, "manifest.json"), "{}");
		fs.writeFileSync(cliPath, `
const fs = require('node:fs');
const path = require('node:path');
fs.mkdirSync(process.env.UNI_OUTPUT_DIR, {recursive:true});
fs.writeFileSync(path.join(process.env.UNI_OUTPUT_DIR,'invocation.json'), JSON.stringify({args:process.argv.slice(2),cwd:process.cwd(),env:Object.fromEntries(['HX_APP_ROOT','UNI_HBUILDERX_PLUGINS','UNI_INPUT_DIR','UNI_OUTPUT_DIR','UNI_APP_X','NODE_PATH'].map(key=>[key,process.env[key]]))}));
const mode = process.env.WMS_WRAPPER_TEST_MODE;
if(mode==='ts') console.log('\\x1b[31merror TS2339: Property missing\\x1b[0m');
else if(mode==='uts') console.error('error UTS1001: Invalid cast');
else if(mode==='kotlin') console.log('e: file:///demo/Index.kt:12: Unresolved reference: missing');
else console.log('warning: Sass deprecation; Build complete');
process.exit(mode==='exit' ? 7 : 0);
`);
		const wrapper = path.join(fixture, "verify.ps1");
		fs.writeFileSync(wrapper, `
param([string]$AppPath,[string]$HxPath)
$ErrorActionPreference='Stop'
$env:HX_APP_ROOT='original-hx'
$env:NODE_PATH='original-modules'
$env:UNI_APP_X='original-flag'
Remove-Item Env:UNI_INPUT_DIR,Env:UNI_OUTPUT_DIR,Env:UNI_HBUILDERX_PLUGINS -ErrorAction SilentlyContinue
$before=(Get-Location).Path
& (Join-Path $AppPath 'scripts/build-hbuilderx.ps1') -HBuilderXPath $HxPath -Platform web
$code=$LASTEXITCODE
$restored = $env:HX_APP_ROOT -eq 'original-hx' -and $env:NODE_PATH -eq 'original-modules' -and $env:UNI_APP_X -eq 'original-flag' -and $null -eq [Environment]::GetEnvironmentVariable('UNI_INPUT_DIR','Process') -and $null -eq [Environment]::GetEnvironmentVariable('UNI_OUTPUT_DIR','Process') -and $null -eq [Environment]::GetEnvironmentVariable('UNI_HBUILDERX_PLUGINS','Process') -and (Get-Location).Path -eq $before
Write-Output "RESTORED=$restored"
exit $code
`);
		for (const [mode, expected] of [["success", 0], ["ts", 1], ["uts", 1], ["kotlin", 1], ["exit", 7]]) {
			const run = spawnSync("pwsh", ["-NoProfile", "-File", wrapper, "-AppPath", app, "-HxPath", hx], { encoding: "utf8", env: { ...process.env, WMS_WRAPPER_TEST_MODE: mode } });
			assert.equal(run.error, undefined);
			assert.equal(run.status, expected, `${mode}: ${run.stdout}\n${run.stderr}`);
			assert.match(run.stdout, /RESTORED=True/);
		}
		const invocation = JSON.parse(fs.readFileSync(path.join(app, "unpackage", "dist", "build", "web", "invocation.json"), "utf8"));
		assert.deepEqual(invocation.args, ["build", "-p", "web"]);
		assert.equal(fs.statSync(invocation.cwd).ino, fs.statSync(app).ino);
		assert.equal(invocation.env.HX_APP_ROOT, hx);
		assert.equal(invocation.env.UNI_APP_X, "true");
		assert.equal(fs.statSync(invocation.env.UNI_INPUT_DIR).ino, fs.statSync(app).ino);
		assert.equal(invocation.env.UNI_HBUILDERX_PLUGINS, path.join(hx, "plugins"));
		assert.equal(invocation.env.NODE_PATH, path.join(hx, "plugins", "uniapp-cli-vite", "node_modules"));
		assert.equal(fs.readdirSync(path.join(app, "unpackage", "verification")).filter(name => name.endsWith(".log")).length, 5);
	} finally {
		const target = path.resolve(fixture);
		assert.equal(path.dirname(target), tempParent);
		assert.ok(path.basename(target).startsWith("wms-build-script-test-"));
		fs.rmSync(target, { recursive: true, force: true });
	}
});
