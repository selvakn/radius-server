# Feature Specification: RADIUS CoA — Admin Session Disconnect

**Feature Branch**: `003-radius-coa-disconnect`
**Created**: 2026-06-27
**Status**: Draft
**Input**: User description:

## Clarifications

### Session 2026-06-27

- Q: Should the disconnect action block the page for up to 5 seconds while waiting for the NAS response, or return immediately? → A: Synchronous — page waits up to 5 seconds, result delivered via redirect flash message
- Q: Where should the "disconnect all sessions" action appear? → A: On the user edit page, alongside enable/disable actions
- Q: Should disconnect attempts be logged to the database? → A: No separate log — session record update (status → stopped) is the only persistent record "add feature for radius incoming, and add the ability to disconnect a user on admin action in the portal"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Disconnect an Active Session from the Admin Portal (Priority: P1)

An administrator views the sessions page and sees a list of currently active PPPoE users. They identify a user that should be disconnected (e.g., unpaid account, abuse, maintenance) and click a disconnect button on that session row. The server immediately sends a Disconnect-Request to the NAS that owns the session. If the NAS accepts, the session disappears from the active list; if the NAS rejects or is unreachable, the admin sees a clear error.

**Why this priority**: Without this, admins can only disable a user in the database and wait for the NAS to naturally expire or re-authenticate the session. Immediate disconnection is essential for abuse response and account suspension.

**Independent Test**: Can be fully tested by establishing a live PPPoE session, navigating to the sessions page, clicking disconnect, and verifying the session is terminated on the NAS within seconds.

**Acceptance Scenarios**:

1. **Given** an active session exists for a user, **When** the admin clicks disconnect on that session row, **Then** the server sends a Disconnect-Request to the NAS and the session is terminated within 5 seconds
2. **Given** the NAS acknowledges the disconnect (Disconnect-ACK), **When** the response is received, **Then** the session status is updated to stopped and a success notice is shown
3. **Given** the NAS rejects the disconnect (Disconnect-NAK) or is unreachable, **When** the response is received or the request times out, **Then** the admin sees a clear error message and the session remains marked active
4. **Given** a session has no NAS IP recorded, **When** the admin attempts to disconnect it, **Then** the disconnect button is disabled or absent

---

### User Story 2 — Disconnect All Sessions for a User (Priority: P2)

An administrator is suspending a user account and wants to terminate all of that user's currently active sessions with a single action, rather than disconnecting each session individually.

**Why this priority**: A user may have multiple simultaneous sessions (or reconnected rapidly); terminating all at once ensures the account suspension takes immediate effect.

**Independent Test**: Can be fully tested by establishing multiple active sessions for the same user, clicking "disconnect all sessions" from the user's edit page, and verifying all sessions are terminated on the NAS.

**Acceptance Scenarios**:

1. **Given** a user has one or more active sessions, **When** the admin clicks "disconnect all" from that user's context, **Then** a Disconnect-Request is sent for each active session and all are terminated
2. **Given** some sessions succeed and some fail, **When** the bulk disconnect completes, **Then** the admin sees a summary indicating which sessions were disconnected and which failed

---

### Edge Cases

- What happens when a Disconnect-Request times out (NAS unreachable or slow to respond)?
- What happens if the session was already terminated on the NAS before the request arrives?
- What if the admin clicks disconnect twice rapidly for the same session?
- How does the server identify the correct session on the NAS when sending the request?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST send a RADIUS Disconnect-Request (RFC 5176) to the NAS when an admin triggers a session disconnect from the portal
- **FR-002**: The Disconnect-Request MUST include the Session-Id and User-Name attributes to uniquely identify the session on the NAS
- **FR-003**: System MUST use the NAS IP from the session record and the shared secret from configuration to construct the Disconnect-Request
- **FR-004**: System MUST wait synchronously for a Disconnect-ACK or Disconnect-NAK response (up to 5 seconds), then redirect the admin with a flash message showing the result
- **FR-005**: On Disconnect-ACK, system MUST mark the session as stopped in the local database and show the admin a success notice
- **FR-006**: On Disconnect-NAK or timeout, system MUST show the admin an error message and leave the session status unchanged
- **FR-007**: The sessions page MUST display a disconnect button for each active session that has a known NAS IP
- **FR-008**: The admin portal MUST provide a "disconnect all sessions" action on the user edit page, alongside the enable/disable controls
- **FR-009**: The disconnect action MUST be protected by the existing CSRF mechanism

### Key Entities

- **DisconnectRequest**: A one-shot outbound RADIUS packet sent to a NAS to terminate a specific session, identified by session ID and username
- **DisconnectResult**: The outcome of a disconnect attempt — success, rejected by NAS, or timed out

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Admin can disconnect an active session within 3 portal interactions from the sessions page
- **SC-002**: Session is terminated on the NAS within 5 seconds of the admin clicking disconnect (when NAS is reachable)
- **SC-003**: Admin receives clear feedback (success or error) within 6 seconds of initiating a disconnect
- **SC-004**: Disconnect button is absent for sessions with no NAS IP, preventing meaningless actions

## Assumptions

- The NAS (MikroTik, EdgeRouter, etc.) supports RFC 5176 Dynamic Authorization and listens on UDP port 3799
- The same shared secret used for RADIUS authentication is used for Disconnect-Request packets to the NAS
- The NAS IP stored in the session record is reachable from the RADIUS server
- A single shared secret applies to all NAS devices (consistent with existing architecture)
- RADIUS CoA (Change of Authorization) for changing session parameters is out of scope — only full disconnect is required
- The disconnect operation is best-effort; if the NAS does not respond, the local session record is not automatically updated
- No separate audit log for disconnect attempts — the session status change (active → stopped on success) is the only persistent record
- Concurrent disconnect requests for the same session are safe — duplicate requests are handled gracefully by the NAS
