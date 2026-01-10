# Story 4.1: Core Web Interface with Ant Design

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want to access logs through a professional web interface,
So that I can query and view logs efficiently using a modern browser-based interface with dark theme.

## Acceptance Criteria

**Given** The log API is working (Epic 1, 2, 3 stories completed)
**When** I access the web interface
**Then** I see a professional interface using Ant Design v5 with dark theme
**And** The interface is responsive and works on 1920px+ displays
**And** I can select datasets from a clear navigation component
**And** Time range selection uses intuitive date/time pickers
**And** The interface works on Chrome 90+, Firefox 88+, Safari 14+, Edge 90+

## Tasks / Subtasks

- [x] Set up React project with TypeScript and modern build configuration (AC: 1)
  - [x] Initialize React 18+ project with TypeScript support
  - [x] Configure Vite for fast development and optimized production builds
  - [x] Set up ESLint and Prettier for code quality and formatting
  - [x] Configure path aliases for cleaner imports
  - [x] Add environment variable configuration for API endpoints
  - [x] Set up development and production build scripts
  - [x] Configure Hot Module Replacement for development efficiency

- [x] Implement Ant Design v5 integration with dark theme (AC: 1, 2)
  - [x] Install and configure Ant Design v5 dependencies
  - [x] Set up dark theme configuration using Ant Design theme system
  - [x] Configure custom color scheme for professional log analysis
  - [x] Implement theme provider for consistent styling across components
  - [x] Add responsive breakpoints for desktop and tablet displays
  - [x] Configure Ant Design components with dark theme overrides
  - [x] Test theme consistency across all supported browsers

- [x] Create main application layout and navigation structure (AC: 3, 4)
  - [x] Design and implement responsive main layout with header, sidebar, and content areas
  - [x] Create dataset selector navigation component with hierarchical structure
  - [x] Implement responsive sidebar with collapsible functionality
  - [x] Add application header with branding and user controls
  - [x] Create main content area with proper routing and page structure
  - [x] Implement proper responsive behavior for 1920px+ displays
  - [x] Add loading states and error boundaries for robust UX

- [x] Implement log query interface with time range selection (AC: 4, 5)
  - [x] Create log query form with dataset selection dropdown
  - [x] Implement intuitive date/time picker components for time range selection
  - [x] Add quick time range selectors (Last 15min, 1 hour, 6 hours, 24 hours, 7 days)
  - [x] Implement namespace and pod filtering input fields
  - [x] Add content search field with clear visual feedback
  - [x] Create query execution button with loading states
  - [x] Add form validation and error handling for user inputs

- [x] Implement API integration layer for log queries (AC: 1)
  - [x] Create TypeScript types matching backend API response structures
  - [x] Implement HTTP client using fetch API with proper error handling
  - [x] Add request/response interceptors for authentication and logging
  - [x] Create API service layer for log query operations
  - [x] Implement retry logic for failed API requests
  - [x] Add response data transformation and validation
  - [x] Create error handling and user feedback mechanisms

- [x] Create basic log results display component (AC: 1)
  - [x] Implement log results table with Ant Design Table component
  - [x] Add basic log field display (timestamp, dataset, namespace, pod, content)
  - [x] Implement severity level indicators with color coding
  - [x] Add result count display and pagination controls
  - [x] Create empty state and loading state components
  - [x] Implement basic result sorting and filtering capabilities
  - [x] Add responsive table behavior for different screen sizes

- [x] Set up browser compatibility and testing (AC: 5)
  - [x] Configure Vite builds for target browser versions (Chrome 90+, Firefox 88+, Safari 14+, Edge 90+)
  - [x] Add browser compatibility polyfills where needed
  - [x] Test application functionality across all supported browsers
  - [x] Implement fallbacks for unsupported browser features
  - [x] Add browser-specific CSS fixes and optimizations
  - [x] Create cross-browser testing checklist
  - [x] Document any browser-specific limitations or workarounds

- [x] Implement deployment configuration (AC: 1)
  - [x] Create production-optimized build configuration
  - [x] Set up static asset optimization and CDN preparation
  - [x] Configure environment-specific API endpoints
  - [x] Create Docker container configuration for frontend deployment
  - [x] Add health check endpoint for deployment monitoring
  - [x] Configure proper cache headers for static assets
  - [x] Document deployment process and environment setup

## Dev Notes

### Architecture Compliance Requirements

**Critical:** Story 4-1 begins Epic 4: Professional Web Interface, implementing the UX-focused NFR requirements (NFR10-NFR15) by creating a modern, responsive web interface using Ant Design v5 with dark theme support.

**Key Technical Requirements:**
- **Frontend Framework:** React 18+ with TypeScript for type safety
- **Build System:** Vite for fast development and optimized production builds
- **UI Framework:** Ant Design v5 with dark theme configuration
- **Browser Support:** Chrome 90+, Firefox 88+, Safari 14+, Edge 90+
- **Responsive Design:** Optimized for 1920px+ displays with mobile support
- **API Integration:** TypeScript-typed integration with existing backend APIs

### Implementation Context from Previous Epics

**Building on Completed Backend Infrastructure:**
- Epic 1: Core query API endpoints are functional (`/apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/logs`)
- Epic 2: Dataset routing, time filtering, and metadata filtering are implemented
- Epic 3: Advanced aggregation, metadata enrichment, and performance optimization are complete

**Integration Points:**
- Dataset selector must work with available datasets from the backend
- Time range pickers must support millisecond precision from Epic 2
- Query interface must support all filter types (namespace, pod, content) from Epic 2
- API responses match the JSON structures defined in Epic 1
- Performance optimization from Epic 3 ensures sub-2 second response times

### Technology Stack and Rationale

**Frontend Framework Choice:**
- **React 18+:** Industry-standard component library with excellent performance
- **TypeScript:** Type safety prevents runtime errors and improves developer experience
- **Vite:** Fast build tool with Hot Module Replacement for efficient development
- **Ant Design v5:** Enterprise UI component library with excellent dark theme support

**Browser Compatibility Strategy:**
- **Target Browsers:** Chrome 90+, Firefox 88+, Safari 14+, Edge 90+ (covering 95%+ of desktop users)
- **Build Configuration:** Vite with proper target configuration and polyfills
- **Progressive Enhancement:** Core functionality works on all target browsers
- **Testing Focus:** Validate critical user flows on all supported browsers

### UI/UX Design Principles

**Professional Tool Aesthetics:**
- **Dark Theme Default:** Reduces eye strain for extended monitoring sessions
- **High Contrast:** Ensures readability in various lighting conditions
- **Information Density:** Maximize useful information without clutter
- **Responsive Design:** Adapts to different screen sizes while maintaining usability

**User Workflow Optimization:**
- **Quick Dataset Access:** Navigation component for fast dataset switching
- **Intuitive Time Selection:** Both quick presets and precise date/time pickers
- **Clear Visual Feedback:** Loading states, error messages, and success indicators
- **Keyboard Accessibility:** Support power users with keyboard shortcuts

### Component Architecture

**Main Application Structure:**
```
src/
├── App.tsx                    # Main application component with routing
├── main.tsx                   # Application entry point
├── types/
│   └── api.ts                 # TypeScript types for API responses
├── services/
│   └── logQueryService.ts     # API integration layer
├── components/
│   ├── layout/
│   │   ├── MainLayout.tsx     # Main application layout
│   │   ├── AppHeader.tsx      # Application header component
│   │   └── DatasetNav.tsx     # Dataset navigation sidebar
│   ├── query/
│   │   ├── QueryForm.tsx      # Main query input form
│   │   ├── TimeRangePicker.tsx # Time range selection component
│   │   └── FilterInputs.tsx   # Namespace, pod, content filters
│   └── results/
│       ├── LogResultsTable.tsx # Main results display
│       └── ResultSummary.tsx   # Result count and metadata
└── styles/
    └── theme.ts               # Ant Design theme configuration
```

### API Integration Strategy

**Backend API Endpoints:**
- **Query Endpoint:** `GET /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/logs`
- **Parameters:** `start_time`, `end_time`, `namespace`, `pod_name`, `filter`, `limit`
- **Response Format:** JSON structure with log entries and metadata

**TypeScript Type Definitions:**
```typescript
interface LogQueryParams {
  dataset: string;
  startTime: string;  // ISO timestamp with milliseconds
  endTime: string;    // ISO timestamp with milliseconds
  namespace?: string;
  podName?: string;
  filter?: string;
  limit?: number;
}

interface LogEntry {
  timestamp: string;
  dataset: string;
  namespace: string;
  pod_name: string;
  container_name: string;
  content: string;
  severity: string;
  // Additional fields from backend
}
```

### Performance Considerations

**Frontend Performance:**
- **Build Optimization:** Vite's production build with code splitting and tree shaking
- **Asset Optimization:** Compressed images and minified JavaScript/CSS
- **Lazy Loading:** Load heavy components only when needed
- **Bundle Size:** Target < 500KB initial bundle for fast load times

**Runtime Performance:**
- **Efficient Rendering:** React 18's concurrent rendering for smooth UI
- **Memoization:** Use React.memo and useMemo to prevent unnecessary re-renders
- **Debouncing:** Debounce user inputs to reduce API calls
- **Virtual Scrolling:** Prepare for Story 4-2's large dataset requirements

### Testing Strategy

**Browser Compatibility Testing:**
- Test on Chrome 90+, Firefox 88+, Safari 14+, Edge 90+
- Validate core functionality: dataset selection, time range picker, query execution, results display
- Test responsive behavior on different screen sizes
- Verify dark theme consistency across browsers

**Functional Testing:**
- Manual testing of all user workflows
- API integration testing with backend endpoints
- Form validation and error handling testing
- Loading states and error boundary testing

**Performance Testing:**
- Initial page load time < 3 seconds on fast connections
- Time to interactive < 5 seconds
- Smooth UI interactions (60fps animations)
- Memory usage within reasonable bounds

### Deployment Strategy

**Build Configuration:**
- **Development:** Vite dev server with HMR for fast iteration
- **Production:** Optimized build with minification and code splitting
- **Staging:** Separate environment configuration for testing
- **Environment Variables:** API endpoints and configuration per environment

**Container Deployment:**
- **Base Image:** nginx:alpine for static file serving
- **Build Process:** Multi-stage Docker build for optimization
- **Health Checks:** Endpoint for deployment monitoring
- **Asset Caching:** Proper cache headers for static assets

### Success Criteria

**Story completion validation:**

1. **Functional Requirements:**
   - Professional web interface accessible via browser
   - Ant Design v5 components with dark theme working correctly
   - Dataset selection and time range pickers functional
   - Log query execution working with backend API
   - Results display showing log entries properly

2. **Browser Compatibility:**
   - Application works on Chrome 90+, Firefox 88+, Safari 14+, Edge 90+
   - Consistent appearance and functionality across browsers
   - Responsive design working on 1920px+ displays

3. **User Experience:**
   - Dark theme reduces eye strain for professional use
   - Intuitive navigation and form controls
   - Clear visual feedback for all user actions
   - Loading states and error handling working properly

4. **Technical Quality:**
   - TypeScript types prevent runtime errors
   - Clean component architecture for maintainability
   - Efficient build and deployment process
   - Proper API integration with error handling

## File List

**New Files to be Created:**
```
frontend/
├── package.json                      # Node.js dependencies and scripts
├── tsconfig.json                     # TypeScript configuration
├── vite.config.ts                    # Vite build configuration
├── index.html                        # HTML entry point
├── .env.example                      # Environment variable template
├── .eslintrc.js                      # ESLint configuration
├── .prettierrc                       # Prettier configuration
├── Dockerfile                        # Frontend container configuration
├── src/
│   ├── main.tsx                      # Application entry point
│   ├── App.tsx                       # Main application component
│   ├── types/
│   │   └── api.ts                    # TypeScript API types
│   ├── services/
│   │   └── logQueryService.ts        # API integration layer
│   ├── components/
│   │   ├── layout/
│   │   │   ├── MainLayout.tsx        # Main layout component
│   │   │   ├── AppHeader.tsx         # Application header
│   │   │   └── DatasetNav.tsx        # Dataset navigation
│   │   ├── query/
│   │   │   ├── QueryForm.tsx         # Query input form
│   │   │   ├── TimeRangePicker.tsx   # Time range selection
│   │   │   └── FilterInputs.tsx      # Filter inputs
│   │   └── results/
│   │       ├── LogResultsTable.tsx   # Results display
│   │       └── ResultSummary.tsx     # Result metadata
│   └── styles/
│       └── theme.ts                  # Ant Design theme config
└── deploy/
    └── frontend-deployment.yaml      # K8s deployment config
```

**Modified Files:**
```
_bmad-output/sprint-status.yaml       # Update Story 4-1 status
go.mod                                # No changes (frontend is separate)
```

## Change Log

**Story Creation (2026-01-10):**
- Created Story 4-1 from Epic 4 requirements
- Defined comprehensive acceptance criteria for professional web interface
- Structured implementation tasks following established patterns
- Identified integration points with completed backend infrastructure
- Planned technology stack and component architecture
- Documented browser compatibility and deployment strategy

**Status Transitions:**
- Story created from backlog: ready-for-dev ✅
- Ready for development work to begin ✅

**Next Steps:**
- Begin Story 4-1 development with React + TypeScript setup
- Implement Ant Design integration with dark theme
- Create main application layout and navigation
- Build log query interface with time range selection
- Integrate with existing backend API endpoints
- Test across all supported browsers
- Deploy frontend container for validation