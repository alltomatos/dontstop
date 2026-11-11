const { spawnSync } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const CHROME = process.env.CHROME_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const BASE = process.env.CONSOLE_URL || 'http://127.0.0.1:3001';
const PROVIDER_ID = process.env.PROVIDER_ID || '';
const LOCALES = ['en', 'pt-PT', 'pt-BR', 'es', 'fr', 'de'];

const SHOTS = [
  { slug: 'provider-catalog', path: '/providers/catalog', size: '1440,1000' },
  { slug: 'provider-configure', path: '/providers/configure/groq', size: '1440,1000' },
  ...(PROVIDER_ID ? [{ slug: 'provider-models', path: `/providers/${PROVIDER_ID}`, size: '1440,2200' }] : [])
];

if (!fs.existsSync(CHROME)) {
  console.error(`Chrome not found at ${CHROME}. Set CHROME_PATH.`);
  process.exit(1);
}
if (!PROVIDER_ID) {
  console.warn('PROVIDER_ID is not set; skipping provider-models screenshot.');
}

let failures = 0;
for (const locale of LOCALES) {
  for (const shot of SHOTS) {
    const outDir = path.join(ROOT, 'docs', 'user-guide', 'assets', locale);
    fs.mkdirSync(outDir, { recursive: true });
    const out = path.join(outDir, `${shot.slug}.png`);
    const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'dontstop-guide-'));
    const [route, hash] = shot.path.split('#');
    const separator = route.includes('?') ? '&' : '?';
    const url = `${BASE}${route}${separator}lang=${encodeURIComponent(locale)}${hash ? `#${hash}` : ''}`;
    const result = spawnSync(CHROME, [
      '--headless=new',
      '--disable-gpu',
      '--hide-scrollbars',
      '--no-first-run',
      '--no-default-browser-check',
      '--disable-background-networking',
      '--disable-sync',
      '--disable-features=Translate',
      `--window-size=${shot.size}`,
      '--force-device-scale-factor=1',
      '--virtual-time-budget=6000',
      `--user-data-dir=${profile}`,
      `--screenshot=${out}`,
      url
    ], { encoding: 'utf8', timeout: 60000 });
    const ok = fs.existsSync(out) && fs.statSync(out).size > 5000;
    if (!ok) {
      failures += 1;
      console.error(`FAIL ${locale}/${shot.slug}`, result.stderr?.split('\n').slice(-2).join(' '));
    } else {
      console.log(`OK ${locale}/${shot.slug} (${fs.statSync(out).size} bytes)`);
    }
    fs.rmSync(profile, { recursive: true, force: true });
  }
}

console.log(`Captured ${LOCALES.length * SHOTS.length - failures}/${LOCALES.length * SHOTS.length} screenshots.`);
if (failures) process.exit(1);
