# Contract: Monthly Usage UI

## Users Page Changes

Two new columns added to the users table (between "Online" and "Actions"):

| Column | Source | Format |
|--------|--------|--------|
| Upload | current month bytes_in per user | fmtbytes (e.g. "1.2 GB") or "—" |
| Download | current month bytes_out per user | fmtbytes or "—" |

## User Edit Page Changes

A monthly history table appended below the existing form:

**Table columns**: Month (e.g. "Jun 2026"), Upload, Download
**Ordering**: Most recent month first
**Max rows**: 24
**Empty state**: "no usage history" if no sessions exist
