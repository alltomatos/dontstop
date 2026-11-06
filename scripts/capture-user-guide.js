const { spawnSync } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const CHROME = process.env.CHROME_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const BASE = process.env.CONSOLE_URL || 'http://127.0.0.1:8765/design-system/mockups/console.html';
const LOCALES = ['en', 'pt-PT', 'pt-BR', 'es', 'fr', 'de'];
const PAGES = ['dashboard', 'providers', 'endpoints', 'api-manager', 'mcp', 'cache', 'dlp', 'observability', 'cost', 'settings'];
const EXTRA = [
  { slug: 'new-api-key', hash: 'api-manager', query: '&open=new-key' },
  { slug: 'onboarding', hash: 'dashboard', query: '&onboarding=1' }
];

if (!fs.existsSync(CHROME)) {
  console.error(`Chrome not found at ${CHROME}. Set CHROME_PATH.`);
  process.exit(1);
}

const jobs = [];
for (const locale of LOCALES) {
  for (const slug of PAGES) {
    jobs.push({ locale, slug, url: `${BASE}?lang=${encodeURIComponent(locale)}#${slug}` });
  }
  for (const item of EXTRA) {
    jobs.push({ locale, slug: item.slug, url: `${BASE}?lang=${encodeURIComponent(locale)}${item.query}#${item.hash}` });
  }
}

let failures = 0;
for (const job of jobs) {
  const outDir = path.join(ROOT, 'docs', 'user-guide', 'assets', job.locale);
  fs.mkdirSync(outDir, { recursive: true });
  const out = path.join(outDir, `${job.slug}.png`);
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'dontstop-shot-'));
  const result = spawnSync(CHROME, [
    '--headless=new',
    '--disable-gpu',
    '--hide-scrollbars',
    '--no-first-run',
    '--no-default-browser-check',
    '--disable-background-networking',
    '--disable-sync',
    '--disable-features=Translate',
    '--window-size=1440,1200',
    '--force-device-scale-factor=1',
    '--virtual-time-budget=4000',
    `--user-data-dir=${profile}`,
    `--screenshot=${out}`,
    job.url
  ], { encoding: 'utf8', timeout: 30000 });
  const ok = fs.existsSync(out) && fs.statSync(out).size > 1000;
  if (!ok) {
    failures += 1;
    console.error(`FAIL ${job.locale}/${job.slug}`, result.stderr?.split('\n').slice(-2).join(' '));
  } else {
    console.log(`OK ${job.locale}/${job.slug} (${fs.statSync(out).size} bytes)`);
  }
  fs.rmSync(profile, { recursive: true, force: true });
}

console.log(`Captured ${jobs.length - failures}/${jobs.length} screenshots.`);
if (failures) process.exit(1);
