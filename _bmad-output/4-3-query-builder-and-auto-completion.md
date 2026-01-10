# Story 4.3: Query Builder and Auto-completion

Status: ready-for-dev

## Story

As an operator,
I want intelligent assistance when building log queries,
So that I can quickly create complex queries without memorizing field names and syntax.

## Acceptance Criteria

**Given** High-performance log display is working (Story 4-2 completed)
**When** I build queries using the interface
**Then** Auto-completion suggests available namespaces, pod names, and field values
**And** Query builder provides visual filter construction for complex conditions
**And** Quick filter buttons for severity levels (Error, Warning, Info, Debug)
**And** Time range shortcuts for common periods (15min, 1hour, today, yesterday)
**And** Field suggestions are based on actual data in the selected dataset

## Tasks / Subtasks

- [ ] Create auto-completion service for field suggestions (AC: 1, 5)
  - [ ] Implement API service for fetching unique namespaces
  - [ ] Implement API service for fetching unique pod names
  - [ ] Implement API service for fetching unique container names
  - [ ] Add debouncing for suggestion requests
  - [ ] Cache suggestions to reduce API calls

- [ ] Implement AutoComplete components for query fields (AC: 1)
  - [ ] Create namespace AutoComplete with suggestions
  - [ ] Create pod name AutoComplete with namespace filtering
  - [ ] Create container name AutoComplete with pod filtering
  - [ ] Add loading states for suggestion fetching
  - [ ] Handle empty results and error states

- [ ] Add quick filter buttons for severity levels (AC: 3)
  - [ ] Create severity filter button group component
  - [ ] Implement buttons for Error, Warning, Info, Debug, Notice
  - [ ] Add toggle behavior (single select or multi-select)
  - [ ] Integrate with QueryForm component
  - [ ] Add visual feedback for active filters

- [ ] Enhance time range picker with shortcuts (AC: 4)
  - [ ] Add predefined time range buttons
  - [ ] Implement Last 15min, 1hour, 6hours, 24hours, 7days shortcuts
  - [ ] Add "Today" and "Yesterday" shortcuts
  - [ ] Add custom date range picker as fallback
  - [ ] Format display relative time (e.g., "Last 15 minutes")

- [ ] Create visual query builder component (AC: 2)
  - [ ] Design filter condition card component
  - [ ] Implement AND/OR logic for combining conditions
  - [ ] Add remove button for each condition
  - [ ] Add "Add Condition" button
  - [ ] Support multiple filter types: equals, contains, regex, exists

- [ ] Integrate with existing QueryForm component (AC: 1-5)
  - [ ] Add AutoComplete to namespace and pod inputs
  - [ ] Replace basic severity select with quick filter buttons
  - [ ] Add time range shortcuts to TimeRangePicker
  - [ ] Ensure backward compatibility with existing query interface
  - [ ] Test integration with VirtualizedLogList

- [ ] Add keyboard shortcuts and accessibility (AC: 2)
  - [ ] Implement Ctrl+Space for auto-complete trigger
  - [ ] Add Escape key to close suggestion dropdown
  - [ ] Support arrow key navigation in suggestions
  - [ ] Add ARIA labels for screen readers
  - [ ] Test with keyboard-only navigation

## Dev Notes

### Architecture Compliance Requirements

**Critical:** Story 4-3 builds on Stories 4-1 and 4-2, adding intelligent query assistance using Ant Design components.

**Key Technical Requirements:**
- **Auto-complete:** Ant Design AutoComplete component with async data fetching
- **Debouncing:** 300ms delay for suggestion requests to avoid API spam
- **Caching:** LRU cache for suggestions (max 100 entries, 5min TTL)
- **Quick Filters:** Button Group component with toggle behavior
- **Time Shortcuts:** Predefined ranges with dayjs calculations

### Implementation Context from Previous Stories

**Building on Completed Work:**
- Story 4-1: QueryForm.tsx with Form.Item components
- Story 4-1: TimeRangePicker.tsx with DatePicker.RangePicker
- Story 4-1: FilterInputs.tsx for namespace, pod, content filters
- Story 4-2: VirtualizedLogList for results display
- Story 4-2: HighlightText for keyword highlighting

**Integration Points:**
- Extend existing FilterInputs.tsx with AutoComplete
- Enhance TimeRangePicker.tsx with shortcut buttons
- Add severity filter buttons to QueryForm.tsx
- Maintain compatibility with existing logQueryService API

### Technology Stack

**Ant Design Components:**
- `AutoComplete` - For async suggestions
- `Button.Group` - For quick filter buttons
- `DatePicker.RangePicker` - Enhanced with shortcuts
- `Select` - For visual query builder operators

**Utilities:**
- `lodash.debounce` - For debouncing suggestion requests
- `dayjs` - For time range calculations

### Component Architecture

**New Components to Create:**
```
src/components/query/
├── AutoCompleteInput.tsx       # Reusable auto-complete wrapper
├── SeverityQuickFilter.tsx      # Quick severity filter buttons
├── TimeRangeShortcuts.tsx       # Predefined time range buttons
└── QueryBuilder.tsx             # Visual filter construction (optional)
```

**Components to Modify:**
```
src/components/query/
├── QueryForm.tsx               # Add severity filters
├── TimeRangePicker.tsx         # Add shortcuts
└── FilterInputs.tsx            # Add AutoComplete
```

**New Services:**
```
src/services/
└── suggestionService.ts         # API for field suggestions
```

### API Integration

**Suggestion Endpoints:**
```typescript
// Namespace suggestions
GET /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/namespaces

// Pod suggestions (filtered by namespace)
GET /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/pods?namespace={ns}

// Container suggestions (filtered by namespace/pod)
GET /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/containers?namespace={ns}&pod={pod}
```

**Response Format:**
```typescript
interface SuggestionResponse {
  values: string[]  // Unique values for the field
  total: number     // Total count
}
```

### Performance Considerations

**Debouncing Strategy:**
- 300ms delay for typing debounce
- 500ms delay for dropdown open
- Cancel pending requests on new input

**Caching Strategy:**
- Map-based cache keyed by dataset + field
- Max 100 entries per field type
- 5 minute TTL for cache entries

### Success Criteria

**Story completion validation:**

1. **Functional Requirements:**
   - Auto-complete suggestions appear when typing
   - Suggestions are based on actual dataset data
   - Quick severity filters work correctly
   - Time range shortcuts set correct date ranges
   - Visual query builder creates valid queries

2. **User Experience:**
   - Smooth interaction with no lag
   - Clear visual feedback for active filters
   - Intuitive keyboard navigation
   - Error handling for failed suggestions

3. **Technical Quality:**
   - Debounced API calls
   - Cached suggestions
   - Clean component architecture
   - Proper TypeScript types

## File List

**New Files to be Created:**
```
frontend/src/components/query/
├── AutoCompleteInput.tsx
├── SeverityQuickFilter.tsx
├── TimeRangeShortcuts.tsx
└── QueryBuilder.tsx (optional)

frontend/src/services/
└── suggestionService.ts
```

**Files to be Modified:**
```
frontend/
├── package.json                    # Add lodash.debounce
├── src/App.tsx                     # Potential updates
└── src/components/query/
    ├── QueryForm.tsx               # Add severity filters
    ├── TimeRangePicker.tsx         # Add shortcuts
    └── FilterInputs.tsx            # Add AutoComplete
```

## Change Log

**Story Creation (2026-01-10):**
- Created Story 4-3 from Epic 4 requirements
- Defined acceptance criteria for auto-completion and query building
- Structured implementation tasks following previous story patterns
- Identified Ant Design components for implementation
- Planned API integration for field suggestions

**Status Transitions:**
- Story created from backlog: ready-for-dev ✅
- Ready for development work to begin ✅
