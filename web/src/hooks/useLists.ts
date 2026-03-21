import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '../db/database'

export function useLists() {
  const lists = useLiveQuery(() => db.lists.orderBy('position').toArray())
  return { lists: lists ?? [], loading: lists === undefined }
}
