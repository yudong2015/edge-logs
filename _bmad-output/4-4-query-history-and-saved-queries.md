# Story 4.4: Query History and Saved Queries

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want to reuse previous queries and save frequently used searches,
So that I can efficiently repeat common troubleshooting workflows without rebuilding queries.

## Acceptance Criteria

**Given** Query builder is implemented (Story 4-3 completed)
**When** I perform queries and want to reuse them
**Then** My last 10 queries are automatically saved and accessible
**And** I can replay any historical query with one click
**And** Query history persists across browser sessions using local storage
**And** I can bookmark frequently used queries for quick access
**And** Saved queries include both parameters and readable descriptions

## Tasks / Subtasks

- [ ] Create query history storage service (AC: 1, 3)
  - [ ] Implement local storage wrapper for query history
  - [ ] Add history persistence with automatic saving on query execution
  - [ ] Implement history management (add, remove, clear)
  - [ ] Add history size limit (max 10 entries with LRU eviction)
  - [ ] Handle local storage errors gracefully

- [ ] Create query history display component (AC: 1, 2)
  - [ ] Design collapsible history panel with list view
  - [ ] Create QueryHistoryItem component for each entry
  - [ ] Display query parameters in human-readable format
  - [ ] Add timestamp for each history entry
  - [ ] Implement click-to-replay functionality

- [ ] Implement saved queries feature (AC: 4, 5)
  - [ ] Create saved query storage with user-defined names
  - [ ] Add bookmark button to query form
  - [ ] Create save query dialog for naming
  - [ ] Implement saved queries management UI
  - [ ] Add edit/delete saved queries functionality

- [ ] Create query serialization/deserialization utilities (AC: 2, 5)
  - [ ] Implement LogQueryParams to/from string conversion
  - [ ] Create human-readable query description generator
  - [ ] Add query comparison for duplicate detection
  - [ ] Handle edge cases (empty filters, all time ranges)

- [ ] Integrate with QueryForm component (AC: 1-5)
  - [ ] Add history panel to query form layout
  - [ ] Sync form state with selected history entry
  - [ ] Auto-save queries on successful execution
  - [ ] Add visual indicators for saved vs history entries
  - [ ] Ensure keyboard navigation support

- [ ] Add local storage fallback and error handling (AC: 3)
  - [ ] Detect local storage availability
  - [ ] Implement memory fallback for private browsing
  - [ ] Add quota exceeded handling
  - [ ] Show user-friendly error messages
  - [ ] Test cross-session persistence

## Dev Notes

### Architecture Compliance Requirements

**Critical:** Story 4-4 builds on Stories 4-1, 4-2, and 4-3, adding query history and saved queries functionality using browser local storage.

**Key Technical Requirements:**
- **Local Storage:** Use browser localStorage for cross-session persistence
- **History Limit:** Maximum 10 entries with FIFO/LRU eviction policy
- **Saved Queries:** Unlimited user-named bookmarks
- **Serialization:** Convert LogQueryParams to/from JSON with description
- **Error Handling:** Graceful degradation when storage unavailable

### Implementation Context from Previous Stories

**Building on Completed Work:**
- Story 4-1: QueryForm.tsx with Form.Item components and state management
- Story 4-1: LogQueryParams type interface for query parameters
- Story 4-2: VirtualizedLogList for results display
- Story 4-3: QueryForm with SeverityQuickFilter and AutoComplete filters
- Story 4-3: TimeRangePicker with Today/Yesterday shortcuts

**Integration Points:**
- Extend QueryForm.tsx with history/saved queries panel
- Use existing LogQueryParams type for serialization
- Maintain compatibility with existing query execution flow
- Reuse existing Ant Design components for UI consistency

### Technology Stack

**Ant Design Components:**
- `Collapse` - For collapsible history/saved panels
- `List` - For displaying history entries
- `Modal` - For save query dialog
- `Input` - For query name input
- `Button` - For bookmark, replay, delete actions
- `Tooltip` - For action button hints
- `Tag` - For query type indicators (history vs saved)
- `Empty` - For empty state display

**Browser APIs:**
- `localStorage` - For cross-session persistence
- `JSON.stringify/parse` - For query serialization

### Component Architecture

**New Components to Create:**
```
src/components/query/
├── QueryHistory.tsx              # History panel with list view
├── SavedQueries.tsx              # Saved queries management
├── QueryHistoryItem.tsx          # Individual history entry
├── SaveQueryDialog.tsx           # Modal for naming saved queries
└── QueryHistoryPanel.tsx         # Combined history + saved panel
```

**New Services:**
```
src/services/
└── queryHistoryService.ts        # History/storage management
```

**New Types:**
```
src/types/
└── queryHistory.ts               # History/saved query types
```

**Components to Modify:**
```
src/components/query/
└── QueryForm.tsx                 # Add history panel integration

src/
└── App.tsx                       # Pass history context if needed
```

### Data Structures

**Query History Entry:**
```typescript
interface QueryHistoryEntry {
  id: string                      // Unique identifier
  timestamp: number               // Unix timestamp
  params: LogQueryParams          // Query parameters
  description: string             // Human-readable description
  resultCount?: number            // Results from this query (optional)
}
```

**Saved Query Entry:**
```typescript
interface SavedQueryEntry extends QueryHistoryEntry {
  name: string                    // User-defined name
  createdAt: number               // Creation timestamp
  updatedAt: number               // Last update timestamp
}
```

**Storage Structure:**
```typescript
interface QueryHistoryStorage {
  history: QueryHistoryEntry[]     // Max 10 entries
  saved: SavedQueryEntry[]         // User bookmarks
  version: string                  // Storage format version
}
```

### Local Storage Keys

```typescript
const STORAGE_KEYS = {
  HISTORY: 'edge-logs-query-history',
  SAVED: 'edge-logs-saved-queries',
  VERSION: 'edge-logs-storage-version',
} as const
```

### Query Description Generation

**Description Format:**
```
"[Time Range] + [Severity] + [Namespace/Pod] + [Content]"
```

**Examples:**
- "Last 1 hour | Error | namespace: default | filter: timeout"
- "Today | All severities | pod: my-app-*"
- "Last 15 minutes | Warning, Error | filter: connection refused"

**Implementation:**
```typescript
function generateQueryDescription(params: LogQueryParams): string {
  const parts: string[] = []

  // Time range description
  const timeDiff = dayjs(params.endTime).diff(params.startTime, 'minutes')
  if (timeDiff < 60) {
    parts.push(`Last ${timeDiff} minutes`)
  } else if (timeDiff < 1440) {
    parts.push(`Last ${Math.round(timeDiff / 60)} hours`)
  } else {
    parts.push(`Last ${Math.round(timeDiff / 1440)} days`)
  }

  // Severity
  parts.push(params.severity || 'All severities')

  // Namespace/Pod
  if (params.namespace) {
    parts.push(`namespace: ${params.namespace}`)
  }
  if (params.podName) {
    parts.push(`pod: ${params.podName}`)
  }

  // Content filter
  if (params.filter) {
    parts.push(`filter: ${params.filter}`)
  }

  return parts.join(' | ')
}
```

### Service Implementation Pattern

```typescript
export class QueryHistoryService {
  private readonly HISTORY_KEY = 'edge-logs-query-history'
  private readonly SAVED_KEY = 'edge-logs-saved-queries'
  private readonly MAX_HISTORY = 10

  // Add to history
  addToHistory(params: LogQueryParams, resultCount?: number): void

  // Get history entries (most recent first)
  getHistory(): QueryHistoryEntry[]

  // Remove from history
  removeFromHistory(id: string): void

  // Clear all history
  clearHistory(): void

  // Save query with name
  saveQuery(params: LogQueryParams, name: string): void

  // Get saved queries
  getSavedQueries(): SavedQueryEntry[]

  // Delete saved query
  deleteSavedQuery(id: string): void

  // Update saved query
  updateSavedQuery(id: string, updates: Partial<SavedQueryEntry>): void
}
```

### Error Handling Strategy

**Storage Errors:**
1. Detect private browsing mode (localStorage access throws)
2. Fall back to in-memory storage
3. Show non-intrusive notification (toast, not alert)
4. Disable history features gracefully

**Quota Exceeded:**
1. Catch QuotaExceededError
2. Prune old history entries
3. Retry save operation
4. Show warning if still fails

**Corrupted Data:**
1. Wrap JSON.parse in try-catch
2. Reset to empty state on parse error
3. Log error for debugging

### UI Layout Integration

**Query Form Layout:**
```
+------------------------------------------+
| Query Form                               |
+------------------------------------------+
| [Time Range]                             |
| [Severity Filters]                       |
| [Filter Inputs]                          |
| [Search] [Reset]                         |
+------------------------------------------+
| Query History & Saved Queries  [Toggle]  |
| +--------------------------------------+  |
| | [History] [Saved Queries]           |  |
| |--------------------------------------|  |
| | [🔍] Last 1 hour | Error | 10m ago  |  |
| |     42 results                        |  |
| |--------------------------------------|  |
| | [★] Production Errors - Today        |  |
| |     Saved 2 days ago                  |  |
| +--------------------------------------+  |
+------------------------------------------+
```

### Accessibility Features

**Keyboard Navigation:**
- `Ctrl+H` / `Cmd+H` - Toggle history panel
- `Ctrl+S` / `Cmd+S` - Save current query
- Arrow keys to navigate history
- Enter to replay selected query
- Delete to remove history entry

**ARIA Labels:**
- `aria-label="Query history panel"`
- `aria-label="Saved queries panel"`
- `aria-label="Replay query: {description}"`
- `aria-label="Delete saved query: {name}"`

### Success Criteria

**Story completion validation:**

1. **Functional Requirements:**
   - Last 10 queries automatically saved on execution
   - History persists across browser sessions
   - Click-to-replay works for all history entries
   - Saved queries with custom names work correctly
   - Delete/edit operations work as expected

2. **User Experience:**
   - History panel collapsible to save space
   - Query descriptions are clear and helpful
   - Visual distinction between history and saved
   - Smooth animations for panel expand/collapse
   - No jank when loading from local storage

3. **Data Persistence:**
   - Survives browser close/reopen
   - Survives page refresh
   - Handles private browsing gracefully
   - No data loss on quota exceeded

## File List

**New Files Created:**
```
frontend/src/components/query/
├── QueryHistoryItem.tsx            # Individual history/saved entry component
├── SaveQueryDialog.tsx             # Modal for naming saved queries
└── QueryHistoryPanel.tsx           # Combined history + saved panel with tabs

frontend/src/services/
└── queryHistoryService.ts          # History/storage management service

frontend/src/types/
└── queryHistory.ts                 # History/saved query type definitions
```

**Files Modified:**
```
frontend/src/components/query/
└── QueryForm.tsx                   # Added history panel integration
```

**Note:** Original plan called for separate `QueryHistory.tsx` and `SavedQueries.tsx` components.
During implementation, these were combined into `QueryHistoryPanel.tsx` with tabbed interface,
which provides better UX and code organization.

## Change Log

**Story Creation (2026-01-10):**
- Created Story 4-4 from Epic 4 requirements
- Defined acceptance criteria for query history and saved queries
- Structured implementation tasks following previous story patterns
- Identified local storage for persistence
- Planned UI integration with QueryForm component

**Status Transitions:**
- Story created from backlog: ready-for-dev ✅
- Ready for development work to begin ✅
