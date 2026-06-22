# Feature Specification: RADIUS Server

**Feature Branch**: `001-radius-server`  
**Created**: 2026-06-22  
**Status**: Draft  
**Input**: User description: "the core functionality of the project is to build a radius server (for authentication - mainly for PPPoE dialup users, from router/firewall like mikrotik), and should be able too add users, remove/disable users, set additional parameters for bandwidth restrictions, etc. Use sqlite for storing the users and user parameters and use golang for the backend, with minimal admin UI. Admin users should be configurable with simple yaml file configurations. Should be distributable as a single binary for easy deployment."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Authenticate PPPoE Dial-up Users (Priority: P1)

A network administrator deploys the RADIUS server and connects it to their MikroTik router. When dial-up users attempt to authenticate via PPPoE, the RADIUS server validates credentials and returns accept/reject responses. The server can also return bandwidth restriction attributes in the authentication response.

**Why this priority**: Core value proposition — without authentication capability, the product serves no purpose.

**Independent Test**: Can be fully tested by configuring a network device to point at the server, attempting PPPoE authentication with valid and invalid credentials, and verifying correct accept/reject responses are returned.

**Acceptance Scenarios**:

1. **Given** a user exists with valid credentials, **When** the RADIUS server receives an Access-Request, **Then** the server responds with Access-Accept
2. **Given** a user does not exist or has invalid credentials, **When** the RADIUS server receives an Access-Request, **Then** the server responds with Access-Reject
3. **Given** a disabled user attempts to authenticate, **When** the RADIUS server receives an Access-Request, **Then** the server responds with Access-Reject
4. **Given** accepted user has bandwidth restrictions configured, **When** the server sends Access-Accept, **Then** the response includes the correct bandwidth attribute (MikroTik Rate-Limit or equivalent)

---

### User Story 2 - Manage User Accounts via Admin UI (Priority: P1)

An administrator accesses a minimal web interface to create new users, disable or permanently remove existing users, and update bandwidth parameters. The interface is keyboard-efficient and presents dense information without superfluous UI elements.

**Why this priority**: Administrators need a straightforward way to manage the user base that drives authentication decisions.

**Independent Test**: Can be fully tested by navigating to the admin UI, creating users, editing their parameters, disabling them, and verifying changes are reflected immediately in subsequent authentication requests.

**Acceptance Scenarios**:

1. **Given** the admin UI is loaded, **When** an administrator submits a new user with username and password, **Then** the user appears in the user list and can authenticate
2. **Given** a user exists, **When** an administrator disables the user from the UI, **Then** the user cannot authenticate via RADIUS
3. **Given** a user exists, **When** an administrator removes the user from the UI, **Then** the user is permanently deleted and cannot authenticate
4. **Given** a user exists, **When** an administrator sets bandwidth limits on the user, **Then** subsequent authentication responses include the configured bandwidth attributes

---

### User Story 3 - Configure Admin Access via File (Priority: P2)

An administrator configures which accounts have admin UI access by editing a YAML configuration file. Changes take effect on next server restart or reload without requiring database migration.

**Why this priority**: Simplifies admin user management and enables configuration-as-practices for deployments where the UI is rarely accessed.

**Independent Test**: Can be fully tested by modifying the configuration file, restarting or reloading the server, and verifying that the configured accounts can access the admin UI.

**Acceptance Scenarios**:

1. **Given** an admin account is listed in the configuration file, **When** that account logs into the admin UI, **Then** access is granted
2. **Given** an account is not listed in the configuration file, **When** that account attempts to log into the admin UI, **Then** access is denied

---

### Edge Cases

- What happens when the server receives a RADIUS request with a malformed packet?
- How does the system handle concurrent authentication requests for the same user?
- What happens when the configuration file contains invalid syntax?
- How does the server behave if the database becomes unreadable or corrupted?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept RADIUS Access-Request packets and respond with Access-Accept or Access-Reject per RFC 2865 over UDP port 1812
- **FR-002**: System MUST support PAP and CHAP authentication methods for PPPoE dial-up clients
- **FR-003**: System MUST allow administrators to create new users with username, password, and optional bandwidth parameters via a web-based admin interface
- **FR-004**: System MUST allow administrators to disable users so they are rejected during authentication without being deleted
- **FR-005**: System MUST allow administrators to permanently remove users from the system
- **FR-006**: System MUST allow administrators to set per-user bandwidth restriction attributes (download/upload rate limits) included in RADIUS Access-Accept responses
- **FR-007**: System MUST include vendor-specific RADIUS attributes for rate limiting when responding to MikroTik-compatible network devices
- **FR-008**: System MUST persist user data in a local, self-contained database with no external dependencies
- **FR-009**: System MUST require login via a form-based authentication flow with server-side sessions; administrator accounts are defined in a configuration file rather than the user database
- **FR-010**: System MUST be distributable as a single executable binary with all dependencies bundled

### Key Entities

- **User**: Dial-up account with username, credentials, enabled/disabled status, and optional bandwidth attributes (max download rate, max upload rate)
- **Admin**: Administrator account with username and credentials, defined outside the user database in a configuration file
- **Bandwidth Profile**: Per-user restrictions specifying maximum download and upload speeds

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: RADIUS Access-Request processed and response sent within 500ms under normal load
- **SC-002**: System supports 1,000 concurrent authentication requests without response degradation beyond acceptable thresholds
- **SC-003**: Administrator can create a new user in fewer than 3 interactions from the admin UI
- **SC-004**: Server starts and begins serving authentication requests within 5 seconds of launch
- **SC-005**: Configuration file changes are applied cleanly on server restart with no data loss

## Assumptions

- Network devices sending RADIUS requests support standard RADIUS protocol (RFC 2865) and UDP port 1812
- A single shared secret protects all RADIUS clients; any connected network device uses the same secret
- Admin UI accesses are made from trusted networks; no OAuth or external identity providers required
- User passwords are stored securely and never in plaintext
- RADIUS Accounting (usage tracking) is out of scope for this initial release — only authentication is supported
- Maximum managed dial-up user count is 100 (small-site deployment; no pagination or bulk operations required)
- Configuration file uses YAML format for admin account definitions and shared secret
