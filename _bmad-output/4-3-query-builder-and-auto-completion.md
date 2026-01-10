# Story 4.3: Query Builder and Auto-completion

Status: review

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

- [x] Create auto-completion service for field suggestions (AC: 1, 5)
  - [x] Implement API service for fetching unique namespaces
  - [x] Implement API service for fetching unique pod names
  - [x] Implement API service for fetching unique container names
  - [x] Add debouncing for suggestion requests
  - [x] Cache suggestions to reduce API calls

- [x] Implement AutoComplete components for query fields (AC: 1)
  - [x] Create namespace AutoComplete with suggestions
  - [x] Create pod name AutoComplete with namespace filtering
  - [x] Create container name AutoComplete with pod filtering
  - [x] Add loading states for suggestion fetching
  - [x] Handle empty results and error states

- [x] Add quick filter buttons for severity levels (AC: 3)
  - [x] Create severity filter button group component
  - [x] Implement buttons for Error, Warning, Info, Debug, Notice
  - [x] Add toggle behavior (single select or multi-select)
  - [x] Integrate with QueryForm component
  - [x] Add visual feedback for active filters

- [x] Enhance time range picker with shortcuts (AC: 4)
  - [x] Add predefined time range buttons
  - [x] Implement Last 15min, 1hour, 6hours, 24hours, 7days shortcuts
  - [x] Add "Today" and "Yesterday" shortcuts
  - [x] Add custom date range picker as fallback
  - [x] Format display relative time (e.g., "Last 15 minutes")

- [x] Create visual query builder component (AC: 2)
  - [x] Design filter condition card component
  - [x] Implement AND/OR logic for combining conditions
  - [x] Add remove button for each condition
  - [x] Add "Add Condition" button
  - [x] Support multiple filter types: equals, contains, regex, exists

- [x] Integrate with existing QueryForm component (AC: 1-5)
  - [x] Add AutoComplete to namespace and pod inputs
  - [x] Replace basic severity select with quick filter buttons
  - [x] Add time range shortcuts to TimeRangePicker
  - [x] Ensure backward compatibility with existing query interface
  - [x] Test integration with VirtualizedLogList

- [x] Add keyboard shortcuts and accessibility (AC: 2)
  - [x] Implement Ctrl+Space for auto-complete trigger
  - [x] Add Escape key to close suggestion dropdown
  - [x] Support arrow key navigation in suggestions
  - [x] Add ARIA labels for screen readers
  - [x] Test with keyboard-only navigation

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

**New Files Created:**
```
frontend/src/components/query/
├── AutoCompleteInput.tsx
├── SeverityQuickFilter.tsx
└── QueryBuilder.tsx

frontend/src/services/
└── suggestionService.ts

frontend/src/components/query/
└── index.ts                         # Component exports
```

**Files Modified:**
```
frontend/
├── src/types/api.ts                 # Added SuggestionResponse type
└── src/components/query/
    ├── QueryForm.tsx               # Added severity quick filter
    ├── TimeRangePicker.tsx         # Added Today/Yesterday shortcuts
    └── FilterInputs.tsx            # Added AutoComplete to filters
```

## Change Log

**Story Creation (2026-01-10):**
- Created Story 4-3 from Epic 4 requirements
- Defined acceptance criteria for auto-completion and query building
- Structured implementation tasks following previous story patterns
- Identified Ant Design components for implementation
- Planned API integration for field suggestions

**Implementation Complete (2026-01-10):**
- Created suggestionService.ts with debouncing and LRU cache
- Created AutoCompleteInput.tsx reusable component with loading states
- Created SeverityQuickFilter.tsx with color-coded buttons
- Created QueryBuilder.tsx for visual filter construction
- Enhanced TimeRangePicker.tsx with Today/Yesterday shortcuts
- Updated FilterInputs.tsx to use AutoComplete for namespace, pod, container
- Updated QueryForm.tsx to integrate SeverityQuickFilter
- Added SuggestionResponse type to api.ts
- Created index.ts for component exports

**Status Transitions:**
- Story created from backlog: ready-for-dev ✅
- Implementation completed: review ✅
