# Feature Specification: Monthly Data Usage

**Feature Branch**: `004-monthly-data-usage`
**Created**: 2026-06-28
**Status**: Draft
**Input**: User description:

## Clarifications

### Session 2026-06-28

- Q: Should monthly boundaries use the server's local timezone (TZ env var) or UTC? → A: Server local timezone — consistent with how timestamps are displayed in the admin UI "build the ability to capture and store the monthly data consumption (upload and downloads), and show the current months usage in the main users page, show the historical monthly usage in the user details page"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — View Current Month Usage on Users Page (Priority: P1)

An administrator opens the main users page and immediately sees each user's data consumption for the current calendar month alongside their account status and bandwidth limits. This allows the admin to quickly spot heavy users or users approaching any informal quota without navigating away.

**Why this priority**: The users page is the primary landing screen after login. Adding at-a-glance monthly usage there maximises the value of each visit.

**Independent Test**: Can be fully tested by generating RADIUS accounting traffic for several users, navigating to the users page, and verifying that each user row shows an upload and download figure that matches the sum of their accounting records for the current month.

**Acceptance Scenarios**:

1. **Given** a user has RADIUS sessions with recorded byte counts in the current calendar month, **When** the admin views the users page, **Then** that user's row shows their total upload and download for the current month in a human-readable unit (MB or GB)
2. **Given** a user has no sessions in the current month, **When** the admin views the users page, **Then** that user's row shows zero or a dash for upload and download
3. **Given** the month changes (e.g., from June to July), **When** the admin views the users page in the new month, **Then** the figures reset to show only the new month's traffic

---

### User Story 2 — View Historical Monthly Usage on User Details Page (Priority: P1)

An administrator opens a specific user's edit/detail page and sees a table of past monthly usage, showing how much that user uploaded and downloaded each month. This supports billing review, dispute resolution, and capacity planning.

**Why this priority**: Historical data is the core audit trail that makes monthly data tracking valuable beyond real-time monitoring.

**Independent Test**: Can be fully tested by simulating sessions across multiple months, navigating to a user's detail page, and verifying that each past month appears as a row with the correct total upload and download figures.

**Acceptance Scenarios**:

1. **Given** a user has accounting records spanning multiple calendar months, **When** the admin views that user's detail page, **Then** a monthly usage table shows one row per month (most recent first) with upload and download totals
2. **Given** a user has no historical data beyond the current month, **When** the admin views the detail page, **Then** only the current month row is shown (or an empty state if the current month has no data)
3. **Given** a month has partial data (sessions started in that month), **When** the admin views the history, **Then** the figures shown are the sum of all completed and ongoing sessions for that month

---

### Edge Cases

- What happens when sessions span a month boundary (started in June, still active in July)?
- How are sessions with zero byte counts handled?
- What is the earliest month displayed in the history (is there a limit)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST aggregate upload and download bytes from session records, grouped by calendar month, for each user
- **FR-002**: The users page MUST display each user's total upload and total download for the current calendar month
- **FR-003**: Usage figures on the users page MUST be formatted in human-readable units (MB or GB), showing a dash when there is no usage
- **FR-004**: The user detail (edit) page MUST display a historical monthly usage table with one row per calendar month, ordered most recent first
- **FR-005**: Each row in the historical table MUST show the month label (e.g., "Jun 2026"), total upload, and total download
- **FR-006**: For sessions that span a month boundary, bytes are attributed to the month in which the accounting Stop record (or last Interim-Update) was received
- **FR-007**: The history table MUST show at most 24 months of data to prevent unbounded page growth
- **FR-008**: Monthly usage data MUST be derived from existing session records with no separate data entry required from the admin

### Key Entities

- **MonthlyUsage**: Aggregated data consumption per user per calendar month — derived view containing month (year + month number), username, total bytes uploaded, total bytes downloaded
- **Session** (existing): Already stores bytes_in, bytes_out, started_at, updated_at — the source of truth for aggregation

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Admin can see the current month's usage for all users within 2 seconds of loading the users page
- **SC-002**: Admin can see up to 24 months of historical usage for a specific user within 2 seconds of loading the user detail page
- **SC-003**: Usage figures are accurate to within the last completed accounting record — no manual recalculation required
- **SC-004**: The users page layout remains scannable with the addition of two new data columns (upload and download)

## Assumptions

- Monthly boundaries are determined by the calendar month in the server's local timezone (the TZ environment variable), consistent with how timestamps are displayed in the admin UI
- Sessions are attributed to the month of their most recent data update (stopped_at if stopped, updated_at if still active)
- The existing sessions table is the sole data source; no separate monthly summary table is pre-computed
- Data is aggregated on page load — no background job or scheduled rollup is required at this scale (≤100 users)
- History is capped at 24 months; older data remains in the sessions table but is not displayed
- Upload (bytes_in) refers to data sent from the client to the internet; download (bytes_out) refers to data received by the client
