import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { runInNewContext } from 'node:vm';

const source = readFileSync(new URL('../../internal/jsplugin/assets/common.js', import.meta.url), 'utf8');
function page(cached = {}) {
  const attributes = new Map(), styles = new Map(), handlers = new Map();
  const storage = new Map(Object.entries(cached));
  const element = {
    setAttribute: (key, value) => attributes.set(key, value), getAttribute: key => attributes.get(key),
    removeAttribute: key => attributes.delete(key), classList: { add() {}, remove() {} },
    style: { setProperty: (key, value) => styles.set(key, value), removeProperty: key => styles.delete(key) },
  };
  const window = { location: { search: '', pathname: '/', origin: 'http://localhost' }, addEventListener: (key, fn) => handlers.set(key, fn) };
  window.parent = window;
  runInNewContext(source, {
    window, document: { documentElement: element, readyState: 'complete', dispatchEvent() {}, querySelectorAll: () => [], getElementById: () => null, addEventListener() {} },
    localStorage: { getItem: key => storage.get(key), setItem: (key, value) => storage.set(key, value) },
    URLSearchParams, CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options?.detail; } },
    console, setTimeout, clearTimeout, fetch,
  });
  return { attributes, styles, storage, window, send: appearance => handlers.get('message')({ data: { type: 'songloft-theme', theme: 'dark', appearance } }) };
}

test('real common.js accepts only boolean accessibility flags and clears absent old-host inputs', () => {
  const p = page();
  p.send({ navigationStyle: 'capsule', reduceTransparency: true, increaseContrast: false, glassFill: 'rgba(23,23,27,.5)' });
  assert.equal(p.attributes.get('data-reduce-transparency'), 'true');
  assert.equal(p.attributes.get('data-increase-contrast'), 'false');
  assert.equal(p.styles.get('--sl-theme-glass-fill'), 'rgba(23, 23, 27, 0.5)');
  p.send({ reduceTransparency: 'true', increaseContrast: 1 });
  assert.equal(p.attributes.has('data-reduce-transparency'), false);
  assert.equal(p.attributes.has('data-increase-contrast'), false);
  p.send(undefined);
  assert.equal(p.attributes.get('data-navigation-style'), 'standard');
  assert.ok(p.window.SongloftPlugin?.apiGet, 'the public bridge still initializes');
});

test('accessibility is never persisted or restored from another host session', () => {
  const p = page({ 'songloft-theme-appearance': JSON.stringify({ navigationStyle: 'capsule', reduceTransparency: true, increaseContrast: true }) });
  assert.equal(p.attributes.has('data-reduce-transparency'), false);
  assert.equal(p.attributes.get('data-navigation-style'), 'capsule');
  p.send({ navigationStyle: 'capsule', reduceTransparency: true, increaseContrast: true });
  const cached = JSON.parse(p.storage.get('songloft-theme-appearance'));
  assert.equal('reduceTransparency' in cached, false);
  assert.equal('increaseContrast' in cached, false);
  const reopened = page(Object.fromEntries(p.storage));
  assert.equal(reopened.attributes.has('data-reduce-transparency'), false);
  assert.equal(reopened.attributes.has('data-increase-contrast'), false);
});
