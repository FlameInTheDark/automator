import { useCallback, useMemo, useState } from 'react'
import type { Edge, Node } from '@xyflow/react'

export interface PipelineDraftState {
  nodes: Node[]
  edges: Edge[]
  pipelineName: string
  pipelineDescription: string
}

interface UsePipelineDraftHistoryOptions {
  initialDraft: PipelineDraftState
  historyLimit?: number
  sanitizeDraft?: (draft: PipelineDraftState) => PipelineDraftState
  serializeDraft?: (draft: PipelineDraftState) => string
}

interface DraftHistoryState {
  draft: PipelineDraftState
  currentKey: string
  savedDraft: PipelineDraftState
  savedKey: string
  past: PipelineDraftState[]
  future: PipelineDraftState[]
  interactionBase: PipelineDraftState | null
  interactionBaseKey: string | null
}

interface ReplaceDraftOptions {
  clearHistory?: boolean
  markSaved?: boolean
}

const DEFAULT_HISTORY_LIMIT = 100

function cloneDraftState(draft: PipelineDraftState): PipelineDraftState {
  if (typeof globalThis.structuredClone === 'function') {
    return globalThis.structuredClone(draft)
  }

  return JSON.parse(JSON.stringify(draft)) as PipelineDraftState
}

function pushWithLimit(history: PipelineDraftState[], snapshot: PipelineDraftState, limit: number) {
  const nextHistory = history.concat(snapshot)
  if (nextHistory.length <= limit) {
    return nextHistory
  }

  return nextHistory.slice(nextHistory.length - limit)
}

export function usePipelineDraftHistory({
  initialDraft,
  historyLimit = DEFAULT_HISTORY_LIMIT,
  sanitizeDraft = cloneDraftState,
  serializeDraft = JSON.stringify,
}: UsePipelineDraftHistoryOptions) {
  const createSnapshot = useCallback((draft: PipelineDraftState) => {
    return sanitizeDraft(cloneDraftState(draft))
  }, [sanitizeDraft])

  const createKey = useCallback((draft: PipelineDraftState) => {
    return serializeDraft(createSnapshot(draft))
  }, [createSnapshot, serializeDraft])

  const [state, setState] = useState<DraftHistoryState>(() => {
    const draft = cloneDraftState(initialDraft)
    const draftSnapshot = createSnapshot(draft)
    const draftKey = createKey(draft)

    return {
      draft,
      currentKey: draftKey,
      savedDraft: draftSnapshot,
      savedKey: draftKey,
      past: [],
      future: [],
      interactionBase: null,
      interactionBaseKey: null,
    }
  })

  const replaceDraft = useCallback((nextDraft: PipelineDraftState, options: ReplaceDraftOptions = {}) => {
    setState((currentState) => {
      const liveDraft = cloneDraftState(nextDraft)
      const nextSnapshot = createSnapshot(liveDraft)
      const nextKey = createKey(liveDraft)

      return {
        draft: liveDraft,
        currentKey: nextKey,
        savedDraft: options.markSaved ? nextSnapshot : currentState.savedDraft,
        savedKey: options.markSaved ? nextKey : currentState.savedKey,
        past: options.clearHistory ? [] : currentState.past,
        future: options.clearHistory ? [] : currentState.future,
        interactionBase: null,
        interactionBaseKey: null,
      }
    })
  }, [createKey, createSnapshot])

  const commitDraft = useCallback((updater: (draft: PipelineDraftState) => PipelineDraftState) => {
    setState((currentState) => {
      const nextDraft = updater(cloneDraftState(currentState.draft))
      const nextKey = createKey(nextDraft)

      if (nextKey === currentState.currentKey) {
        return {
          ...currentState,
          draft: nextDraft,
          currentKey: nextKey,
          interactionBase: null,
          interactionBaseKey: null,
        }
      }

      return {
        ...currentState,
        draft: nextDraft,
        currentKey: nextKey,
        past: pushWithLimit(currentState.past, createSnapshot(currentState.draft), historyLimit),
        future: [],
        interactionBase: null,
        interactionBaseKey: null,
      }
    })
  }, [createKey, createSnapshot, historyLimit])

  const updateDraftLive = useCallback((updater: (draft: PipelineDraftState) => PipelineDraftState) => {
    setState((currentState) => {
      const nextDraft = updater(cloneDraftState(currentState.draft))

      return {
        ...currentState,
        draft: nextDraft,
        currentKey: createKey(nextDraft),
      }
    })
  }, [createKey])

  const beginInteraction = useCallback(() => {
    setState((currentState) => {
      if (currentState.interactionBaseKey !== null) {
        return currentState
      }

      return {
        ...currentState,
        interactionBase: createSnapshot(currentState.draft),
        interactionBaseKey: currentState.currentKey,
      }
    })
  }, [createSnapshot])

  const commitInteraction = useCallback((updater?: (draft: PipelineDraftState) => PipelineDraftState) => {
    setState((currentState) => {
      const nextDraft = updater ? updater(cloneDraftState(currentState.draft)) : currentState.draft
      const nextKey = createKey(nextDraft)

      if (currentState.interactionBase === null || currentState.interactionBaseKey === null) {
        if (nextKey === currentState.currentKey) {
          return {
            ...currentState,
            draft: nextDraft,
            currentKey: nextKey,
          }
        }

        return {
          ...currentState,
          draft: nextDraft,
          currentKey: nextKey,
          past: pushWithLimit(currentState.past, createSnapshot(currentState.draft), historyLimit),
          future: [],
        }
      }

      if (nextKey === currentState.interactionBaseKey) {
        return {
          ...currentState,
          draft: nextDraft,
          currentKey: nextKey,
          interactionBase: null,
          interactionBaseKey: null,
        }
      }

      return {
        ...currentState,
        draft: nextDraft,
        currentKey: nextKey,
        past: pushWithLimit(currentState.past, currentState.interactionBase, historyLimit),
        future: [],
        interactionBase: null,
        interactionBaseKey: null,
      }
    })
  }, [createKey, createSnapshot, historyLimit])

  const markSaved = useCallback((draftOverride?: PipelineDraftState) => {
    setState((currentState) => {
      const nextDraft = draftOverride ? cloneDraftState(draftOverride) : currentState.draft
      const nextSnapshot = createSnapshot(nextDraft)
      const nextKey = createKey(nextDraft)

      return {
        ...currentState,
        savedDraft: nextSnapshot,
        savedKey: nextKey,
      }
    })
  }, [createKey, createSnapshot])

  const undo = useCallback(() => {
    setState((currentState) => {
      if (currentState.past.length === 0) {
        return currentState
      }

      const previousDraft = currentState.past[currentState.past.length - 1]
      const currentSnapshot = createSnapshot(currentState.draft)

      return {
        ...currentState,
        draft: cloneDraftState(previousDraft),
        currentKey: createKey(previousDraft),
        past: currentState.past.slice(0, -1),
        future: [currentSnapshot, ...currentState.future],
        interactionBase: null,
        interactionBaseKey: null,
      }
    })
  }, [createKey, createSnapshot])

  const redo = useCallback(() => {
    setState((currentState) => {
      if (currentState.future.length === 0) {
        return currentState
      }

      const [nextDraft, ...remainingFuture] = currentState.future
      const currentSnapshot = createSnapshot(currentState.draft)

      return {
        ...currentState,
        draft: cloneDraftState(nextDraft),
        currentKey: createKey(nextDraft),
        past: pushWithLimit(currentState.past, currentSnapshot, historyLimit),
        future: remainingFuture,
        interactionBase: null,
        interactionBaseKey: null,
      }
    })
  }, [createKey, createSnapshot, historyLimit])

  return useMemo(() => ({
    draft: state.draft,
    savedDraft: state.savedDraft,
    replaceDraft,
    commitDraft,
    updateDraftLive,
    beginInteraction,
    commitInteraction,
    markSaved,
    undo,
    redo,
    canUndo: state.past.length > 0,
    canRedo: state.future.length > 0,
    isDirty: state.currentKey !== state.savedKey,
    hasPendingInteraction: state.interactionBaseKey !== null,
  }), [
    beginInteraction,
    commitDraft,
    commitInteraction,
    markSaved,
    redo,
    replaceDraft,
    state.currentKey,
    state.draft,
    state.future.length,
    state.interactionBaseKey,
    state.past.length,
    state.savedDraft,
    state.savedKey,
    undo,
    updateDraftLive,
  ])
}
