import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join, resolve } from 'node:path';
import { test } from 'node:test';

const require = createRequire(import.meta.url);
const root = resolve(import.meta.dirname, '..');
const cache = join(root, 'node_modules', '.cache');
const toolingManifest = require.resolve('@zhengshi/vue-typescript/package.json');
const toolingRequire = createRequire(toolingManifest);
const toolingPackage = JSON.parse(readFileSync(toolingManifest, 'utf8'));
const vueCli = resolve(dirname(toolingManifest), toolingPackage.bin['vue-typecheck']);
const nativePackagePath = require.resolve('typescript/package.json');
const nativePackage = JSON.parse(readFileSync(nativePackagePath, 'utf8'));
const nativeCli = resolve(dirname(nativePackagePath), nativePackage.bin.tsc);
mkdirSync(cache, { recursive: true });

function run(cli, args) {
  const result = spawnSync(process.execPath, [cli, ...args], {
    cwd: root,
    encoding: 'utf8',
    timeout: 30_000,
  });
  assert.ifError(result.error);
  return { status: result.status, output: result.stdout + result.stderr };
}

// Keep generated fixtures outside src; cleanup is limited to this test's directory.
function withFixture(source, verify) {
  const directory = mkdtempSync(join(cache, 'typecheck-contract-'));
  try {
    writeFileSync(join(directory, 'tsconfig.json'), JSON.stringify({
      extends: join(root, 'tsconfig.json'),
      compilerOptions: { types: [], noEmit: true },
      include: ['*.ts', '*.vue'],
      vueCompilerOptions: { strictTemplates: true },
    }));
    writeFileSync(join(directory, 'Child.vue'), `<script setup lang="ts">
const props = defineProps<{ count: number }>();
</script>
<template>{{ props.count.toFixed(2) }}</template>`);
    writeFileSync(join(directory, 'ModelChild.vue'), `<script setup lang="ts">
const props = defineProps<{ modelValue: number }>();
const emit = defineEmits<{ 'update:modelValue': [value: number] }>();
</script>
<template><button @click="emit('update:modelValue', props.modelValue + 1)">{{ props.modelValue }}</button></template>`);
    writeFileSync(join(directory, 'App.vue'), source.vue ?? '<template>OK</template>');
    writeFileSync(join(directory, 'plain.ts'), source.ts ?? 'export {};');
    verify(directory);
  } finally {
    assert.equal(dirname(resolve(directory)), cache);
    rmSync(directory, { recursive: true, force: true });
  }
}

test('the default compiler is official TypeScript 7 and Vue uses its native checker adapter', () => {
  assert.equal(nativePackage.name, 'typescript');
  assert.match(nativePackage.version, /^7\./);
  assert.match(run(nativeCli, ['--version']).output, /Version 7\./);
  const adapter = toolingRequire('typescript/package.json');
  assert.equal(adapter.name, 'typescript-native-bridge');
  assert.match(adapter.version, /\.tsgo\.7\./);
  assert.equal(adapter.version.split('.tsgo.')[1], nativePackage.version,
    'Upgrade the native compiler and Vue adapter together');
});

test('valid TS, Vue templates, component props and v-model pass', () => {
  withFixture({ vue: `<script setup lang="ts">
import { ref } from 'vue';
import Child from './Child.vue';
import ModelChild from './ModelChild.vue';
const count = ref(1);
</script>
<template><Child :count="count" /><ModelChild v-model="count" />{{ count.toFixed(2) }}</template>` }, (directory) => {
    const result = run(vueCli, ['--noEmit', '--pretty', 'false', '-p', directory]);
    assert.equal(result.status, 0, result.output);
    assert.match(result.output, /TNB ACTIVE/);
  });
});

const invalidCases = [
  ['Vue script assignment', { vue: '<script setup lang="ts">const count: number = "bad";</script><template>{{ count }}</template>' }, /App\.vue\(1,\d+\): error TS2322/],
  ['Vue template property', { vue: '<script setup lang="ts">const row = { id: 1 };</script><template>{{ row.missing }}</template>' }, /App\.vue\(1,\d+\): error TS2339/],
  ['cross-component prop', { vue: '<script setup lang="ts">import Child from "./Child.vue";</script><template><Child count="bad" /></template>' }, /App\.vue\(1,\d+\): error TS2322/],
  ['default v-model input', { vue: '<script setup lang="ts">import { ref } from "vue"; import ModelChild from "./ModelChild.vue"; const count = ref<string | number>("bad");</script><template><ModelChild v-model="count" /></template>' }, /App\.vue\(1,\d+\): error TS2322/],
  ['strict null in template', { vue: '<script setup lang="ts">import { ref } from "vue"; const row = ref<{ id: number } | null>(null);</script><template>{{ row.id }}</template>' }, /App\.vue\(1,\d+\): error TS18047/],
  ['unused script binding', { vue: '<script setup lang="ts">const unused = 1;</script><template>OK</template>' }, /App\.vue\(1,\d+\): error TS6133/],
  ['plain TypeScript assignment', { ts: 'export const count: number = "bad";' }, /plain\.ts\(1,\d+\): error TS2322/],
];

for (const [name, source, diagnostic] of invalidCases) {
  test(`${name} still fails the Vue typecheck with a source-mapped diagnostic`, () => {
    withFixture(source, (directory) => {
      const result = run(vueCli, ['--noEmit', '--pretty', 'false', '-p', directory]);
      assert.equal(result.status, 2, result.output);
      assert.match(result.output, /TNB ACTIVE/);
      assert.match(result.output, diagnostic);
    });
  });
}

test('official TypeScript 7 rejects invalid TS independently of the Vue adapter', () => {
  withFixture({ ts: 'export const count: number = "bad";' }, (directory) => {
    writeFileSync(join(directory, 'tsconfig.native.json'), JSON.stringify({
      extends: join(root, 'tsconfig.node.json'),
      compilerOptions: { composite: false, incremental: false, noEmit: true, types: [] },
      include: ['plain.ts'],
    }));
    const result = run(nativeCli, ['--noEmit', '--pretty', 'false', '-p', join(directory, 'tsconfig.native.json')]);
    assert.equal(result.status, 1, result.output);
    assert.match(result.output, /plain\.ts\(1,\d+\): error TS2322/);
    assert.doesNotMatch(result.output, /TNB ACTIVE/);
  });
});
