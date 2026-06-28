# Data Model: Monthly Data Usage

## No Schema Changes

No new tables. All data derived from the existing `sessions` table.

## New Derived Type: MonthlyUsage

```
MonthlyUsage:
  Month    string   "YYYY-MM" (e.g. "2026-06")
  BytesIn  int64    total upload bytes for the month
  BytesOut int64    total download bytes for the month
```

## Attribution Rule

For each session, bytes are attributed to the month of the most recent data point:
- Stopped sessions → month of `stopped_at`
- Active sessions → month of `updated_at`

Month determined using server local timezone (`TZ` env var).

## Derived Queries

**GetCurrentMonthUsage**: returns `map[username]MonthlyUsage` for the current calendar month.
**GetMonthlyUsageHistory(username)**: returns `[]MonthlyUsage` ordered newest-first, max 24 entries.
