const fs = require('fs');
const path = require('path');
const { validate, LOCALES } = require('./validate');

const ROOT = path.resolve(__dirname, '..', '..');
const DIST = path.join(ROOT, 'admin', 'dist');

function copy(source, target) {
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.copyFileSync(source, target);
}

try {
  const report = validate();
  fs.rmSync(DIST, { recursive: true, force: true });
  for (const file of ['i18n.js', 'preferences.js', 'source-keys.js', 'guide-links.js', 'console-runtime.js']) {
    copy(path.join(ROOT, 'admin', 'src', file), path.join(DIST, 'src', file));
  }
  for (const locale of LOCALES) {
    copy(path.join(ROOT, 'admin', 'locales', `${locale}.json`), path.join(DIST, 'locales', `${locale}.json`));
  }
  copy(path.join(ROOT, 'docs', 'user-guide', 'manifest.json'), path.join(DIST, 'user-guide', 'manifest.json'));
  fs.writeFileSync(path.join(DIST, 'build-info.json'), JSON.stringify({
    name: 'dontstop-admin-i18n',
    builtAt: new Date().toISOString(),
    ...report
  }, null, 2) + '\n', 'utf8');
  console.log(`OK: build written to admin/dist (${report.docs} guides, ${report.images} images).`);
} catch (error) {
  console.error(`FAIL: ${error.message}`);
  process.exit(1);
}
