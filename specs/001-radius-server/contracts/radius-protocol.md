# Contract: RADIUS Protocol (UDP 1812)

## Overview

The server listens on UDP port 1812 and speaks RFC 2865 RADIUS authentication protocol. Network devices (MikroTik routers, etc.) act as RADIUS clients (NAS).

## Authentication (Access-Request → Access-Accept/Reject)

### Packet Flow

```
NAS ──[UDP/1812]──► RADIUS Server
     Access-Request
     ◄──────────────
     Access-Accept OR Access-Reject
```

### Access-Request Attributes (received from NAS)

| Attribute | Code | Type | Description |
|-----------|------|------|-------------|
| User-Name | 1 | String | PPPoE username |
| User-Password | 2 | Encrypted | PAP password (MD5 encrypted with shared secret) |
| CHAP-Password | 3 | Octets | CHAP response (16 bytes) |
| CHAP-Challenge | 60 | Octets | CHAP challenge value |
| NAS-IP-Address | 4 | Address | NAS IP |
| NAS-Port | 5 | Integer | Physical port |
| Service-Type | 6 | Integer | 2 = Framed |
| Framed-Protocol | 7 | Integer | 1 = PPP |

### Access-Accept Attributes (sent on success)

| Attribute | Code | Type | Value |
|-----------|------|------|-------|
| Framed-Protocol | 7 | Integer | 1 (PPP) |
| Framed-Compression | 13 | Integer | 1 (Van Jacobson) |
| *Mikrotik-Rate-Limit* | VSA 14988/8 | String | `"<down>k/<up>k"` (only if rates configured) |

### Access-Reject Attributes (sent on failure)

| Attribute | Code | Type | Value |
|-----------|------|------|-------|
| Reply-Message | 18 | String | Human-readable reason (optional) |

### Authentication Methods

**PAP** (Password Authentication Protocol):
- Password in `User-Password` attribute, encrypted: `XOR(password, MD5(shared_secret + Request-Authenticator))`
- Server decrypts, then bcrypt-compares against stored hash

**CHAP** (Challenge Handshake Authentication Protocol):
- `CHAP-Password` = `MD5(CHAP-ID + plaintext_password + CHAP-Challenge)`
- Server computes same MD5 using stored plaintext... **Note**: CHAP requires access to the plaintext password server-side, which conflicts with bcrypt storage. For CHAP support, a separate reversibly-encrypted or plaintext password column would be needed. **CHAP support is deferred** — PAP only for initial release.

### Shared Secret

Single global shared secret configured in `config.yaml`. Used for:
- Decrypting PAP User-Password
- Generating Response-Authenticator (HMAC over Access-Accept/Reject)

### Error Handling

| Scenario | Server Action |
|----------|---------------|
| Malformed packet | Silent drop (no response) |
| Unknown NAS (wrong secret) | Silent drop or Access-Reject |
| User not found | Access-Reject |
| User disabled | Access-Reject |
| Invalid password | Access-Reject |
| DB unavailable | Access-Reject (fail-safe) |

## Scope Exclusions

- RADIUS Accounting (port 1813) — out of scope
- RADIUS CoA (Change of Authorization) — out of scope
- Per-NAS shared secrets — out of scope (single global secret)
