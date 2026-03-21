import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '../db/database'

export function useLabels() {
  const labels = useLiveQuery(() => db.labels.orderBy('name').toArray())
  return { labels: labels ?? [], loading: labels === undefined }
}
