import type { ReactNode } from 'react'

export function highlightText(text: string, search: string): ReactNode[] {
  if (!search.trim()) {
    return [text]
  }

  const parts: ReactNode[] = []
  const lowerText = text.toLowerCase()
  const lowerSearch = search.toLowerCase()
  let lastIndex = 0

  let index = lowerText.indexOf(lowerSearch)
  while (index !== -1) {
    if (index > lastIndex) {
      parts.push(text.slice(lastIndex, index))
    }
    parts.push(<mark key={index}>{text.slice(index, index + search.length)}</mark>)
    lastIndex = index + search.length
    index = lowerText.indexOf(lowerSearch, lastIndex)
  }

  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex))
  }

  return parts
}
