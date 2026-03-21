import { compare, type HLCTimestamp } from '../hlc/hlc'

/**
 * Last-Writer-Wins merge. Returns the value with the later HLC timestamp.
 * On equal timestamps, remote wins for deterministic tiebreaking across devices.
 */
export function mergeLWW<T>(
  localVal: T, localHLC: HLCTimestamp,
  remoteVal: T, remoteHLC: HLCTimestamp,
): [T, HLCTimestamp] {
  if (compare(remoteHLC, localHLC) >= 0) {
    return [remoteVal, remoteHLC]
  }
  return [localVal, localHLC]
}
