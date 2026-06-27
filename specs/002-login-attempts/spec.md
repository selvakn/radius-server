# Feature Specification: Login Attempt Log

**Feature Branch**: `002-login-attempts`
**Created**: 2026-06-27
**Status**: Draft
**Input**: User description:

## Clarifications

### Session 2026-06-27

- Q: Should the attempt log row show the outcome of each attempt, or only username, 24h count, and last timestamp? → A: Show a "last outcome" column (accepted / rejected) per username row
- Q: Should the attempt log cap the number of rows shown, or display all unique usernames? → A: Cap at 200 rows, ordered by most recent attempt first
- Q: If the username already exists when the admin submits the pre-populated form, what should happen? → A: Show a notice "user already exists" and redirect to that user's edit page "Build a feature to show the attempts to login (user) credentials. Show it in its own section, show the number of attempts to login in the last 24 hours and the last time it attempted to login. Admin should be able to add the user to allowed users with one click."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — View Login Attempt Log (Priority: P1)

An administrator opens the admin UI and navigates to a dedicated section showing every unique username that has attempted authentication. For each username, the admin can see how many times it attempted in the last 24 hours and when it last attempted. Unknown usernames — those not present in the user database — are visually distinguished, making it easy to spot devices trying to connect that haven't been provisioned yet.

**Why this priority**: Core value — without seeing who is attempting to connect, the admin has no visibility into rejected or unknown devices and cannot act on them.

**Independent Test**: Can be fully tested by sending RADIUS authentication requests (both from known and unknown usernames), navigating to the attempt log section, and verifying that usernames, counts, and last-attempt timestamps are shown correctly.

**Acceptance Scenarios**:

1. **Given** one or more RADIUS authentication requests have been received, **When** the admin views the login attempt log, **Then** each unique username appears as a row with its attempt count for the last 24 hours and the timestamp of its most recent attempt
2. **Given** a username is not in the user database, **When** it appears in the attempt log, **Then** it is visually distinguished from known users (e.g. with a different status label)
3. **Given** all attempts for a username are older than 24 hours, **When** viewing the log, **Then** the 24-hour count shows zero but the last-attempt timestamp is still shown
4. **Given** no authentication attempts have ever been received, **When** the admin views the log, **Then** an empty state message is shown

---

### User Story 2 — Provision User from Attempt Log (Priority: P1)

An administrator sees an unknown username in the attempt log and wants to allow it. They click a single "add" button on that row, which immediately creates a user account with that username and takes the admin to set a password and optional bandwidth limits.

**Why this priority**: The attempt log is most actionable when the admin can convert an unknown username into a provisioned user without navigating away or re-typing the username.

**Independent Test**: Can be fully tested by finding an unknown username in the attempt log, clicking add, completing the user creation form (password and optional rates), and verifying the user is created and subsequent authentication requests with that username are accepted.

**Acceptance Scenarios**:

1. **Given** an unknown username appears in the attempt log, **When** the admin clicks the add button for that row, **Then** the new user form opens pre-populated with that username
2. **Given** the admin completes and submits the pre-populated form, **When** the user is created, **Then** subsequent authentication requests for that username are accepted by the RADIUS server
3. **Given** a username in the attempt log is already in the user database (known), **When** viewing the log, **Then** the add button is not shown for that row (it already exists)
4. **Given** the admin submits the pre-populated new user form but the username was created by another admin in the interim, **When** the duplicate is detected on submit, **Then** the system shows a "user already exists" notice and redirects to that user's edit page

---

### Edge Cases

- What happens when the same username attempts login thousands of times in 24 hours — is there a display cap?
- How does the log behave when the server has been running for less than 24 hours?
- What happens if the admin tries to add a username that was created by another admin between page load and button click? → If the username already exists on form submit, show a notice and redirect to that user's edit page
- How long are attempt records retained before being purged?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST record every RADIUS authentication attempt with the username, timestamp, and outcome (accepted or rejected)
- **FR-002**: System MUST display a login attempt log section in the admin UI, separate from the user list
- **FR-003**: For each unique username in the log, system MUST show the number of attempts in the last 24 hours, the timestamp of the most recent attempt, and the outcome of the most recent attempt (accepted or rejected)
- **FR-004**: System MUST visually distinguish usernames that are not present in the user database from those that are known users
- **FR-005**: System MUST provide a one-click action on unknown username rows that opens the new user form pre-populated with that username
- **FR-006**: Attempt records older than 7 days MUST be automatically purged to prevent unbounded storage growth
- **FR-007**: The log MUST update on each page load to reflect the latest attempts
- **FR-008**: The log MUST display at most 200 unique usernames, ordered by most recent attempt first

### Key Entities

- **AuthAttempt**: A single authentication attempt record with username, timestamp, and outcome (accepted/rejected)
- **AttemptSummary**: A derived view per unique username showing 24-hour count, last attempt time, last attempt outcome (accepted/rejected), and whether the username is a known user

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Admin can identify unknown usernames attempting to connect within 2 page interactions from login
- **SC-002**: Admin can provision an unknown username as a new user in fewer than 3 interactions from the attempt log
- **SC-003**: Attempt log page loads in under 2 seconds with up to 10,000 stored attempt records
- **SC-004**: Attempt records older than 7 days are purged automatically with no manual intervention

## Assumptions

- All RADIUS authentication attempts are recorded regardless of outcome (accepted or rejected)
- The 24-hour window is rolling (last 24 hours from current time), not a calendar day
- Attempt records are stored in the same local database as user accounts
- Automatic purge of records older than 7 days is sufficient retention for this use case — no archiving or export is required
- The attempt log is read-only except for the one-click add action; there is no bulk-delete or manual purge UI
- Ordering is by most recent attempt first (most actionable at the top)
- The log shows all unique usernames ever seen (up to the 200-row cap), not just the last 24 hours — the 24h count may be zero for older usernames
