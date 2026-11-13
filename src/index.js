// 浏览器埋点 SDK：兼容 Go 采集器 /pingd/front/web（旧参数名不变）。
// 用法：
//   <script src="/dist/index.js"></script>
//   <script>siteH5.configure({ pvid: '800800' }); siteH5.pgv();</script>
(function (global) {
	var DEFAULT_PATH = '/pingd/front/web';
	var UVMARK_KEY = 'uvmark';
	var SESSION_KEY = 'nodelogs_session_id';

	var settings = { collectUrl: null, pvid: '800800' };

	function defaultCollectUrl() {
		try {
			if (global.location && global.location.origin) return global.location.origin + DEFAULT_PATH;
		} catch (e) { /* 忽略：退化为相对路径 */ }
		return DEFAULT_PATH;
	}

	function collectUrl() {
		return settings.collectUrl || defaultCollectUrl();
	}

	function storage(kind) {
		try {
			return kind === 'session' ? global.sessionStorage : global.localStorage;
		} catch (e) {
			return null;
		}
	}

	function genId() {
		try {
			var crypto = global.crypto;
			if (crypto && typeof crypto.randomUUID === 'function') {
				return crypto.randomUUID().replace(/-/g, '');
			}
			if (crypto && typeof crypto.getRandomValues === 'function') {
				var bytes = new Uint8Array(16);
				crypto.getRandomValues(bytes);
				var hex = '';
				for (var i = 0; i < bytes.length; i++) {
					hex += ('0' + bytes[i].toString(16)).slice(-2);
				}
				return hex;
			}
		} catch (e) { /* 忽略：退化为 Math.random */ }
		return Math.random().toString(36).slice(2) + Date.now().toString(36) + Math.random().toString(36).slice(2);
	}

	function getOrCreate(kind, key) {
		var store = storage(kind);
		if (store) {
			try {
				var val = store.getItem(key);
				if (val) return val;
				var created = genId();
				try { store.setItem(key, created); } catch (e) { /* 存储不可写：仅本次有效 */ }
				return created;
			} catch (e) { /* 存储不可读：退化为本次有效 */ }
		}
		return genId();
	}

	function safe(fn, fallback) {
		try {
			var v = fn();
			return v === undefined || v === null ? fallback : v;
		} catch (e) {
			return fallback;
		}
	}

	// 每次调用生成全新参数对象：保留全部旧参数名，新增 event_id/session_id。
	function buildParams(overrides) {
		var loc = global.location || {};
		var screen = global.screen || {};
		var nav = global.navigator || {};
		var params = {
			dm: safe(function () { return loc.host; }, ''),
			pvid: settings.pvid,
			uvmark: getOrCreate('local', UVMARK_KEY),
			session_id: getOrCreate('session', SESSION_KEY),
			event_id: genId(),
			url: safe(function () { return loc.pathname; }, ''),
			arg: safe(function () { return loc.search; }, ''),
			ty: 0,
			v: '1.0.0',
			scr: safe(function () { return screen.width + 'x' + screen.height; }, ''),
			scl: safe(function () { return screen.colorDepth + '-bit'; }, ''),
			lg: safe(function () { return (nav.language || nav.userLanguage || '').toLowerCase(); }, ''),
			tz: safe(function () { return new Date().getTimezoneOffset() / 60; }, ''),
			d: Date.now(),
			random: safe(function () { return Math.random().toString().slice(2, 24); }, '')
		};
		if (overrides) {
			for (var k in overrides) {
				if (Object.prototype.hasOwnProperty.call(overrides, k)) params[k] = overrides[k];
			}
		}
		return params;
	}

	function serialize(obj) {
		var str = [];
		for (var p in obj) {
			if (Object.prototype.hasOwnProperty.call(obj, p)) {
				str.push(encodeURIComponent(p) + '=' + encodeURIComponent(obj[p]));
			}
		}
		return str.join('&');
	}

	function sendBeacon(query) {
		try {
			var nav = global.navigator;
			if (nav && typeof nav.sendBeacon === 'function') {
				return nav.sendBeacon(collectUrl() + '?' + query);
			}
		} catch (e) { /* 静默失败 */ }
		return false;
	}

	function sendImage(query) {
		try {
			// fmt=gif：图片 Beacon 兼容场景下服务端返回透明 GIF。
			var img = new global.Image();
			img.onload = img.onabort = img.onerror = function () {
				img.onload = img.onabort = img.onerror = null;
			};
			img.src = collectUrl() + '?' + query + '&fmt=gif';
			return true;
		} catch (e) { /* 静默失败 */ }
		return false;
	}

	function report(overrides) {
		var query = serialize(buildParams(overrides));
		if (!sendBeacon(query)) sendImage(query);
	}

	var siteH5 = {
		configure: function (opts) {
			if (!opts) return;
			if (opts.collectUrl) settings.collectUrl = opts.collectUrl;
			if (opts.pvid) settings.pvid = opts.pvid;
		},
		pgv: function () { report(); },
		clickStat: function (evName) { report({ dm: 'click', url: evName }); }
	};

	try {
		if (typeof global.addEventListener === 'function') {
			global.addEventListener('unload', function () {
				try { sendBeacon(serialize(buildParams({ d: Date.now() }))); } catch (e) { /* 静默失败 */ }
			});
		}
	} catch (e) { /* 忽略 */ }

	global.siteH5 = siteH5;
	// CommonJS 导出仅供 node --test 测试；浏览器 script 引入时 module 未定义，走 global.siteH5。
	if (typeof module !== 'undefined' && module.exports) {
		siteH5.__test__ = { serialize: serialize, buildParams: buildParams, getCollectUrl: collectUrl };
		module.exports = siteH5;
	}
})(typeof window !== 'undefined' ? window : globalThis);
