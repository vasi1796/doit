import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '../db/database'

export function useLabels() {
  // Sort in JS rather than via the position index: rows that predate the
  // position backfill (e.g. redelivered historical LabelCreated events) have
  // no position and would be dropped by orderBy('position').
  const labels = useLiveQuery(() =>
    db.labels.toArray().then((all) =>
      all.sort((a, b) => {
        if (a.position != null && b.position != null && a.position !== b.position) {
          return a.position < b.position ? -1 : 1
        }
        if ((a.position != null) !== (b.position != null)) {
          return a.position != null ? -1 : 1
        }
        return a.name < b.name ? -1 : a.name > b.name ? 1 : 0
      }),
    ),
  )
  return { labels: labels ?? [], loading: labels === undefined }
}
