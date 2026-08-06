#!/usr/bin/env node
//
// responsive-check.mjs — horizontal-overflow regression test for docs/index.html
//
// Loads the docs page in headless Chromium at a set of viewport widths and
// asserts that the *page* never scrolls horizontally, while allowing code
// blocks to scroll inside their own box.
//
// Zero npm dependencies: drives Chromium over the DevTools Protocol using the
// WebSocket client built into Node >= 22.
//
// Usage:
//   node scripts/responsive-check.mjs
//   node scripts/responsive-check.mjs --file=/tmp/other.html --widths=320,390
//   node scripts/responsive-check.mjs --report=/tmp/report.json
//
// Chromium is discovered via $CHROME_BIN, the Playwright cache, or $PATH.

import { spawn } from 'node:child_process';
import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const REPO = path.resolve(HERE, '..');

const args = Object.fromEntries(
  process.argv.slice(2).map((a) => {
    const [k, v = 'true'] = a.replace(/^--/, '').split('=');
    return [k, v];
  }),
);

const FILE = path.resolve(args.file ?? path.join(REPO, 'docs', 'index.html'));
const MOBILE_WIDTHS = (args.widths ?? '320,360,375,390,414,768')
  .split(',')
  .map(Number);
const DESKTOP_WIDTHS = (args.desktop ?? '1024,1280,1440')
  .split(',')
  .map(Number);
const TOL = 0.5; // sub-pixel rounding tolerance
// --inject=<file.css> appends a stylesheet at runtime. Lets candidate fixes be
// measured against the unmodified page before committing to them.
const INJECT_CSS = args.inject ? readFileSync(path.resolve(args.inject), 'utf8') : null;

// ── Chromium discovery ───────────────────────────────────────────────────────

function findChrome() {
  if (process.env.CHROME_BIN) return process.env.CHROME_BIN;

  const pw = path.join(os.homedir(), '.cache', 'ms-playwright');
  if (existsSync(pw)) {
    for (const dir of readdirSync(pw).sort().reverse()) {
      for (const rel of [
        'chrome-linux64/chrome',
        'chrome-linux/chrome',
        'chrome-headless-shell-linux64/chrome-headless-shell',
        'Chromium.app/Contents/MacOS/Chromium',
      ]) {
        const p = path.join(pw, dir, rel);
        if (existsSync(p)) return p;
      }
    }
  }

  for (const p of [
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
    '/usr/bin/google-chrome',
    '/usr/bin/google-chrome-stable',
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  ]) {
    if (existsSync(p)) return p;
  }
  return null;
}

// ── Minimal CDP client ───────────────────────────────────────────────────────

class CDP {
  #ws;
  #id = 0;
  #pending = new Map();
  #listeners = new Set();

  static async connect(url) {
    const c = new CDP();
    c.#ws = new WebSocket(url);
    await new Promise((res, rej) => {
      c.#ws.addEventListener('open', res, { once: true });
      c.#ws.addEventListener('error', rej, { once: true });
    });
    c.#ws.addEventListener('message', (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.id !== undefined) {
        const p = c.#pending.get(msg.id);
        if (!p) return;
        c.#pending.delete(msg.id);
        msg.error ? p.reject(new Error(JSON.stringify(msg.error))) : p.resolve(msg.result);
      } else {
        for (const l of c.#listeners) l(msg);
      }
    });
    return c;
  }

  send(method, params = {}, sessionId) {
    const id = ++this.#id;
    return new Promise((resolve, reject) => {
      this.#pending.set(id, { resolve, reject });
      this.#ws.send(JSON.stringify({ id, method, params, sessionId }));
    });
  }

  once(method, sessionId, timeoutMs = 15000) {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.#listeners.delete(fn);
        reject(new Error(`timeout waiting for ${method}`));
      }, timeoutMs);
      const fn = (msg) => {
        if (msg.method !== method) return;
        if (sessionId && msg.sessionId !== sessionId) return;
        clearTimeout(timer);
        this.#listeners.delete(fn);
        resolve(msg.params);
      };
      this.#listeners.add(fn);
    });
  }

  close() {
    this.#ws.close();
  }
}

async function launchChrome(bin) {
  const userDataDir = await mkdtemp(path.join(os.tmpdir(), 'gccli-respcheck-'));
  const proc = spawn(
    bin,
    [
      '--headless=new',
      '--remote-debugging-port=0',
      `--user-data-dir=${userDataDir}`,
      '--no-sandbox',
      '--disable-gpu',
      '--disable-dev-shm-usage',
      '--no-first-run',
      '--no-default-browser-check',
      '--disable-extensions',
      '--force-device-scale-factor=1',
      // Overlay (zero-width) scrollbars, as on iOS/Android, so the vertical
      // scrollbar cannot skew the horizontal measurement.
      '--hide-scrollbars',
      'about:blank',
    ],
    { stdio: ['ignore', 'pipe', 'pipe'] },
  );

  const wsUrl = await new Promise((resolve, reject) => {
    let buf = '';
    const timer = setTimeout(() => reject(new Error('chrome did not report a devtools endpoint')), 20000);
    proc.stderr.on('data', (d) => {
      buf += d;
      const m = buf.match(/DevTools listening on (ws:\/\/\S+)/);
      if (m) {
        clearTimeout(timer);
        resolve(m[1]);
      }
    });
    proc.on('exit', (code) => reject(new Error(`chrome exited early (${code}): ${buf}`)));
  });

  return {
    wsUrl,
    async kill() {
      proc.kill('SIGKILL');
      await rm(userDataDir, { recursive: true, force: true }).catch(() => {});
    },
  };
}

// ── In-page probe ────────────────────────────────────────────────────────────
//
// Runs inside the page. Returns everything a single viewport measurement needs.

const PROBE = /* js */ `(() => {
  const TOL = ${TOL};
  const vw = window.innerWidth;

  const label = (el) => {
    let s = el.tagName.toLowerCase();
    if (el.id) s += '#' + el.id;
    if (el.classList.length) s += '.' + [...el.classList].filter(c => c !== 'visible').join('.');
    return s;
  };
  const chain = (el) => {
    const parts = [];
    for (let p = el; p && p !== document.documentElement; p = p.parentElement) parts.unshift(label(p));
    return parts.slice(-4).join(' > ');
  };

  // Anti-cheat: the page must not hide the symptom by clipping the root.
  const rootClip = {
    html: getComputedStyle(document.documentElement).overflowX,
    body: getComputedStyle(document.body).overflowX,
  };

  // An element is "excused" if some ancestor below <body> is a horizontal
  // scroll/clip container — that is legitimate self-contained overflow.
  const excused = (el) => {
    for (let p = el.parentElement; p && p !== document.body && p !== document.documentElement; p = p.parentElement) {
      const ox = getComputedStyle(p).overflowX;
      if (ox === 'auto' || ox === 'scroll' || ox === 'hidden' || ox === 'clip') return true;
    }
    return false;
  };

  const offenders = [];
  for (const el of document.body.querySelectorAll('*')) {
    const r = el.getBoundingClientRect();
    if (r.width === 0 && r.height === 0) continue;
    if (r.right <= vw + TOL && r.left >= -TOL) continue;
    if (excused(el)) continue;
    offenders.push({
      sel: chain(el),
      left: +r.left.toFixed(1),
      right: +r.right.toFixed(1),
      width: +r.width.toFixed(1),
      overhang: +(r.right - vw).toFixed(1),
      text: (el.textContent || '').trim().slice(0, 60),
    });
  }
  offenders.sort((a, b) => b.overhang - a.overhang);

  // Code blocks: their own box must fit; long lines may scroll inside it.
  const codeBlocks = [...document.querySelectorAll('pre.code-block')].map((el) => {
    const r = el.getBoundingClientRect();
    return {
      sel: chain(el),
      fits: r.right <= vw + TOL && r.left >= -TOL,
      overflowX: getComputedStyle(el).overflowX,
      whiteSpace: getComputedStyle(el).whiteSpace,
      scrolls: el.scrollWidth > el.clientWidth + TOL,
    };
  });

  // Desktop layout signature — guards against the fix flattening the design.
  const tracks = (sel) => {
    const el = document.querySelector(sel);
    if (!el) return null;
    return getComputedStyle(el).gridTemplateColumns
      .split(' ')
      .map((v) => Math.round(parseFloat(v)))
      .join(' ');
  };
  const signature = {
    '.split': tracks('.split'),
    '.card-grid': tracks('.card-grid'),
    '.features-grid': tracks('.features-grid'),
    '.arch-grid': tracks('.arch-grid'),
    '.examples-grid': tracks('.examples-grid'),
    '.config-grid': tracks('.config-grid'),
    '.hero-grid': tracks('.hero-grid'),
    '.terminal-meta': tracks('.terminal-meta'),
    '.cmd-row': tracks('.cmd-row'),
    heroTitleFontSize: getComputedStyle(document.querySelector('.hero-title')).fontSize,
    sectionPadding: getComputedStyle(document.querySelector('.section')).paddingLeft,
    navHeight: Math.round(document.querySelector('.nav-inner').getBoundingClientRect().height),
  };

  return {
    vw,
    docScrollWidth: document.documentElement.scrollWidth,
    bodyScrollWidth: document.body.scrollWidth,
    rootClip,
    offenders: offenders.slice(0, 12),
    offenderCount: offenders.length,
    codeBlocks,
    signature,
  };
})()`;

const EXPAND_ALL = /* js */ `(() => {
  document.querySelectorAll('.cmd-group').forEach((g) => g.classList.add('is-open'));
  document.querySelectorAll('.fade-in').forEach((e) => e.classList.add('visible'));
  return document.querySelectorAll('.cmd-group.is-open').length;
})()`;

// ── Runner ───────────────────────────────────────────────────────────────────

async function measure(cdp, session, { width, height, mobile, expand }) {
  await cdp.send('Emulation.setDeviceMetricsOverride', {
    width,
    height,
    deviceScaleFactor: 1,
    mobile,
    screenWidth: width,
    screenHeight: height,
  }, session);

  const loaded = cdp.once('Page.loadEventFired', session);
  await cdp.send('Page.navigate', { url: pathToFileURL(FILE).href }, session);
  await loaded;

  // Web fonts are blocked (see Network.setBlockedURLs) so this settles at once.
  await cdp.send('Runtime.evaluate', {
    expression: 'document.fonts.ready.then(() => true)',
    awaitPromise: true,
    returnByValue: true,
  }, session);

  if (INJECT_CSS) {
    await cdp.send('Runtime.evaluate', {
      expression: `(() => {
        const s = document.createElement('style');
        s.textContent = ${JSON.stringify(INJECT_CSS)};
        document.head.appendChild(s);
        return true;
      })()`,
      returnByValue: true,
    }, session);
  }

  if (expand) {
    await cdp.send('Runtime.evaluate', { expression: EXPAND_ALL, returnByValue: true }, session);
  }

  const { result } = await cdp.send('Runtime.evaluate', {
    expression: PROBE,
    returnByValue: true,
  }, session);

  if (args.shot) {
    const { data } = await cdp.send('Page.captureScreenshot', {
      format: 'png',
      captureBeyondViewport: true,
      clip: { x: 0, y: 0, width, height: Number(args.shotHeight ?? 2600), scale: 1 },
    }, session);
    const out = path.join(args.shot, `${args.shotPrefix ?? 'shot'}-${width}${expand ? '-expanded' : ''}.png`);
    await writeFile(out, Buffer.from(data, 'base64'));
  }

  return result.value;
}

async function main() {
  const bin = findChrome();
  if (!bin) {
    console.error('No Chromium found. Set CHROME_BIN=/path/to/chrome and retry.');
    process.exit(2);
  }
  if (!existsSync(FILE)) {
    console.error(`No such file: ${FILE}`);
    process.exit(2);
  }

  const chrome = await launchChrome(bin);
  const cdp = await CDP.connect(chrome.wsUrl);
  const failures = [];
  const report = { file: FILE, chrome: bin, runs: [] };

  try {
    const { targetId } = await cdp.send('Target.createTarget', { url: 'about:blank' });
    const { sessionId } = await cdp.send('Target.attachToTarget', { targetId, flatten: true });

    await cdp.send('Page.enable', {}, sessionId);
    await cdp.send('Runtime.enable', {}, sessionId);
    await cdp.send('Network.enable', {}, sessionId);
    // Determinism: no network, no webfont race. Layout must hold on fallback fonts.
    await cdp.send('Network.setBlockedURLs', {
      urls: ['*fonts.googleapis.com*', '*fonts.gstatic.com*'],
    }, sessionId);

    // mobile:false on purpose. Chrome's mobile emulation "shrinks to fit" when
    // content overflows — it silently widens the layout viewport to the page's
    // min-content width, which makes scrollWidth === innerWidth always true and
    // hides the very bug we are testing for. A plain layout viewport at the
    // target width (plus --hide-scrollbars) measures the real thing.
    const cases = [
      ...MOBILE_WIDTHS.flatMap((width) => [
        { width, height: 844, mobile: false, expand: false },
        { width, height: 844, mobile: false, expand: true },
      ]),
      ...DESKTOP_WIDTHS.flatMap((width) => [
        { width, height: 900, mobile: false, expand: false },
        { width, height: 900, mobile: false, expand: true },
      ]),
    ];

    for (const c of cases) {
      const r = await measure(cdp, sessionId, c);
      const name = `${c.width}px${c.expand ? ' (expanded)' : ''}`;
      const before = failures.length;
      report.runs.push({ ...c, ...r });

      if (r.rootClip.html !== 'visible' || r.rootClip.body !== 'visible') {
        failures.push(
          `${name}: root overflow-x is clipped (html=${r.rootClip.html}, body=${r.rootClip.body}). ` +
            `Overflow must be fixed at the source, not hidden on <html>/<body>.`,
        );
      }

      if (r.docScrollWidth > r.vw + TOL) {
        const top = r.offenders
          .slice(0, 6)
          .map((o) => `      +${o.overhang}px  ${o.sel}  "${o.text}"`)
          .join('\n');
        failures.push(
          `${name}: page scrolls horizontally — scrollWidth ${r.docScrollWidth} > viewport ${r.vw}` +
            `\n    ${r.offenderCount} element(s) cross the right edge:\n${top}`,
        );
      } else if (r.offenderCount > 0) {
        const top = r.offenders
          .slice(0, 6)
          .map((o) => `      +${o.overhang}px  ${o.sel}  "${o.text}"`)
          .join('\n');
        failures.push(`${name}: ${r.offenderCount} element(s) escape the viewport:\n${top}`);
      }

      const clippedCode = r.codeBlocks.filter((c2) => !c2.fits);
      if (clippedCode.length) {
        failures.push(
          `${name}: ${clippedCode.length} code block(s) do not fit the viewport:\n` +
            clippedCode.map((c2) => `      ${c2.sel}`).join('\n'),
        );
      }
      const notScrollable = r.codeBlocks.filter(
        (c2) => c2.overflowX !== 'auto' && c2.overflowX !== 'scroll',
      );
      if (notScrollable.length) {
        failures.push(
          `${name}: ${notScrollable.length} code block(s) are not horizontal scroll containers`,
        );
      }

      console.log(
        `${failures.length === before ? 'ok  ' : 'FAIL'} ${name.padEnd(20)} ` +
          `scrollWidth=${String(r.docScrollWidth).padStart(5)} ` +
          `viewport=${String(r.vw).padStart(5)} offenders=${r.offenderCount}`,
      );
    }

    // At least one code block must actually scroll at 320px — proof that long
    // lines are preserved and scrollable rather than clipped away.
    const narrow = report.runs.find((r) => r.width === MOBILE_WIDTHS[0] && r.expand === false);
    if (narrow && !narrow.codeBlocks.some((c) => c.scrolls)) {
      failures.push(
        `${MOBILE_WIDTHS[0]}px: no code block scrolls internally — long lines may have been ` +
          `wrapped or clipped instead of kept scrollable`,
      );
    }
  } finally {
    cdp.close();
    await chrome.kill();
  }

  if (args.report) {
    await writeFile(args.report, JSON.stringify(report, null, 2));
    console.log(`\nreport written to ${args.report}`);
  }

  console.log('');
  if (failures.length) {
    console.log(`FAIL — ${failures.length} problem(s)\n`);
    for (const f of failures) console.log(`  • ${f}\n`);
    process.exit(1);
  }
  console.log(`PASS — no horizontal page overflow at ${[...MOBILE_WIDTHS, ...DESKTOP_WIDTHS].join(', ')} px`);
}

await main();
