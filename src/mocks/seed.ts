// mock 专用种子随机数：同条件返回稳定数据，不同条件返回不同数据。
// 与 src/mocks/report.ts 的实现思路一致，但抽出来供各新模块共用。

export function hashString(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

export function mulberry32(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/** 由查询上下文生成确定性随机数发生器。 */
export function rngFor(key: string): () => number {
  return mulberry32(hashString(key))
}

export function intBetween(rng: () => number, min: number, max: number): number {
  return min + Math.floor(rng() * (max - min + 1))
}

export function floatBetween(rng: () => number, min: number, max: number, digits = 4): number {
  return Number((min + rng() * (max - min)).toFixed(digits))
}
