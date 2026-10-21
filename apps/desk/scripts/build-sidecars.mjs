// Builds sdd and sdd-desk into src-tauri/binaries/<name>-<target-triple>, the layout Tauri expects for sidecars.
// sdd is bundled as sdd-studio-sdd, so a package does not install a second /usr/bin/sdd; sdd-desk looks for that name.
// Run after `npm run build:desk`, which embeds the UI into sdd-desk.
import { execFileSync } from 'node:child_process';
import { mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

const repo = resolve(import.meta.dirname, '..', '..', '..');
const out = resolve(import.meta.dirname, '..', 'src-tauri', 'binaries');
const triple = process.env.TAURI_ENV_TARGET_TRIPLE || execFileSync('rustc', ['--print', 'host-tuple'], { encoding: 'utf8' }).trim();
const ext = triple.includes('windows') ? '.exe' : '';

const goTarget = {
  'x86_64-unknown-linux-gnu': ['linux', 'amd64'],
  'aarch64-unknown-linux-gnu': ['linux', 'arm64'],
  'x86_64-apple-darwin': ['darwin', 'amd64'],
  'aarch64-apple-darwin': ['darwin', 'arm64'],
  'x86_64-pc-windows-msvc': ['windows', 'amd64'],
  'aarch64-pc-windows-msvc': ['windows', 'arm64'],
}[triple];
if (!goTarget) {
  console.error(`unsupported target ${triple}`);
  process.exit(1);
}

mkdirSync(out, { recursive: true });
for (const [name, cmd] of [['sdd-studio-sdd', 'sdd'], ['sdd-desk', 'sdd-desk']]) {
  const target = resolve(out, `${name}-${triple}${ext}`);
  execFileSync('go', ['build', '-trimpath', '-o', target, `./cmd/${cmd}`], {
    cwd: repo,
    stdio: 'inherit',
    env: { ...process.env, GOOS: goTarget[0], GOARCH: goTarget[1], CGO_ENABLED: '0' },
  });
  console.log(`built ${target}`);
}
