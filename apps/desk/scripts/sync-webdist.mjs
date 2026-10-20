// Copies the built UI into internal/desk/webdist, where `go build ./cmd/sdd-desk` embeds it.
import { cpSync, existsSync, readdirSync, rmSync } from 'node:fs';
import { join, resolve } from 'node:path';

const src = resolve(import.meta.dirname, '..', 'dist', 'desk', 'browser');
const dst = resolve(import.meta.dirname, '..', '..', '..', 'internal', 'desk', 'webdist');
if (!existsSync(src)) {
  console.error(`missing build output ${src}: run ng build first`);
  process.exit(1);
}
for (const entry of readdirSync(dst)) {
  if (entry !== 'README.md') {
    rmSync(join(dst, entry), { recursive: true, force: true });
  }
}
cpSync(src, dst, { recursive: true });
console.log(`UI copied to ${dst}`);
