'use strict';
// SDK 行为测试：零依赖，使用 Node 内置测试运行器。
// 运行：node --test src/sdk.test.js
const { describe, it, afterEach } = require('node:test');
const assert = require('node:assert/strict');

const SDK_PATH = './index.js';

// 构造最小浏览器环境 mock。
function createWindow() {
  const images = [];
  const beacons = [];
  const listeners = {};
  const mkStore = () => {
    const map = {};
    return {
      getItem: (k) => (Object.prototype.hasOwnProperty.call(map, k) ? map[k] : null),
      setItem: (k, v) => { map[k] = String(v); },
      removeItem: (k) => { delete map[k]; },
      __map: map,
    };
  };
  const win = {
    location: { host: 'example.com', pathname: '/home', search: '?a=1', origin: 'https://example.com' },
    screen: { width: 1920, height: 1080, colorDepth: 24 },
    navigator: { language: 'zh-CN' },
    localStorage: mkStore(),
    sessionStorage: mkStore(),
    addEventListener: (type, fn) => { (listeners[type] = listeners[type] || []).push(fn); },
    Image: function Image() { const img = {}; images.push(img); return img; },
    __images: images,
    __beacons: beacons,
    __listeners: listeners,
  };
  win.navigator.sendBeacon = (url) => { beacons.push(String(url)); return true; };
  return win;
}

function loadSDK(win) {
  global.window = win;
  delete require.cache[require.resolve(SDK_PATH)];
  return require(SDK_PATH);
}

afterEach(() => {
  delete global.window;
  delete require.cache[require.resolve(SDK_PATH)];
});

function queryOf(url) {
  const u = new URL(url, 'https://example.com');
  return u.searchParams;
}

describe('siteH5 SDK（Go 采集器兼容）', () => {
  it('无 sendBeacon 时用 Image Beacon 上报全部旧参数', () => {
    const win = createWindow();
    delete win.navigator.sendBeacon;
    const sdk = loadSDK(win);
    sdk.pgv();
    assert.equal(win.__images.length, 1);
    const q = queryOf(win.__images[0].src);
    for (const k of ['pvid', 'uvmark', 'url', 'arg', 'v', 'scr', 'lg', 'tz', 'd', 'ty', 'scl', 'random']) {
      assert.ok(q.get(k) !== null && q.get(k) !== '', `缺失旧参数 ${k}`);
    }
    assert.equal(q.get('pvid'), '800800');
    assert.equal(q.get('url'), '/home');
  });

  it('每次上报携带 event_id 与 session_id；event_id 唯一、session 稳定', () => {
    const win = createWindow();
    delete win.navigator.sendBeacon;
    const sdk = loadSDK(win);
    sdk.pgv();
    sdk.pgv();
    const q1 = queryOf(win.__images[0].src);
    const q2 = queryOf(win.__images[1].src);
    assert.ok(q1.get('event_id'));
    assert.ok(q1.get('session_id'));
    assert.notEqual(q1.get('event_id'), q2.get('event_id'));
    assert.equal(q1.get('session_id'), q2.get('session_id'));
  });

  it('首访生成 uvmark 并持久化，再次加载复用同一值', () => {
    const win = createWindow();
    delete win.navigator.sendBeacon;
    const sdk = loadSDK(win);
    sdk.pgv();
    const first = queryOf(win.__images[0].src).get('uvmark');
    assert.ok(first);
    assert.equal(win.localStorage.getItem('uvmark'), first);
    // 模拟二次加载（同一 localStorage）。
    const sdk2 = loadSDK(win);
    sdk2.pgv();
    assert.equal(queryOf(win.__images[1].src).get('uvmark'), first);
  });

  it('默认上报 Go 兼容路径，configure 可覆盖采集地址与 pvid', () => {
    const win = createWindow();
    delete win.navigator.sendBeacon;
    const sdk = loadSDK(win);
    sdk.pgv();
    assert.ok(win.__images[0].src.includes('/pingd/front/web'));
    sdk.configure({ collectUrl: 'https://collect.example.com/api/v1/collect', pvid: '900900' });
    sdk.pgv();
    assert.ok(win.__images[1].src.startsWith('https://collect.example.com/api/v1/collect?'));
    assert.equal(queryOf(win.__images[1].src).get('pvid'), '900900');
  });

  it('旧版 ty/scl 语义不变，random 每次上报都不同', () => {
    const win = createWindow();
    delete win.navigator.sendBeacon;
    const sdk = loadSDK(win);
    sdk.pgv();
    sdk.pgv();
    const q1 = queryOf(win.__images[0].src);
    const q2 = queryOf(win.__images[1].src);
    assert.equal(q1.get('ty'), '0');
    assert.equal(q1.get('scl'), '24-bit');
    assert.ok(q1.get('random'));
    assert.notEqual(q1.get('random'), q2.get('random'));
  });

  it('clickStat 上报事件名且不污染后续 pgv 的页面参数', () => {
    const win = createWindow();
    delete win.navigator.sendBeacon;
    const sdk = loadSDK(win);
    sdk.clickStat('BTN_CLICK');
    assert.equal(queryOf(win.__images[0].src).get('url'), 'BTN_CLICK');
    sdk.pgv();
    assert.equal(queryOf(win.__images[1].src).get('url'), '/home');
  });

  it('有 sendBeacon 时优先使用 Beacon，unload 也发送 Beacon', () => {
    const win = createWindow();
    const sdk = loadSDK(win);
    sdk.pgv();
    assert.equal(win.__beacons.length, 1);
    assert.ok(queryOf(win.__beacons[0]).get('event_id'));
    for (const fn of win.__listeners.unload || []) fn();
    assert.equal(win.__beacons.length, 2);
  });

  it('Image/Beacon/存储异常时静默失败，不抛异常阻塞页面', () => {
    const win = createWindow();
    win.Image = function () { throw new Error('Image blocked'); };
    win.navigator.sendBeacon = () => { throw new Error('beacon blocked'); };
    win.localStorage = { getItem: () => { throw new Error('denied'); }, setItem: () => { throw new Error('denied'); } };
    win.sessionStorage = { getItem: () => { throw new Error('denied'); }, setItem: () => { throw new Error('denied'); } };
    const sdk = loadSDK(win);
    assert.doesNotThrow(() => { sdk.pgv(); sdk.clickStat('X'); });
    for (const fn of win.__listeners.unload || []) assert.doesNotThrow(fn);
  });
});
