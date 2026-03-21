import { HLCClock } from '../hlc/hlc'

/** Single HLC clock instance shared by all operations. */
export const clock = new HLCClock()
