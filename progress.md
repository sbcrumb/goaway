# GoAway Custom Fork — Progress

## Branch: `custom-clients`
Fork of [pommee/goaway](https://github.com/pommee/goaway), personal customizations on top of upstream.

---

## Done

### Bug Fixes (upstreamable)
- **DNS SERVFAIL on local hostnames** — `handleStandardQuery` was hardcoding `RcodeServerFailure` when an upstream query returned no answers (e.g. AAAA query for an A-only host). Now correctly passes through the upstream Rcode (NOERROR, NXDOMAIN, etc.).
- **Client hostname showing "unknown"** — `reverseDNSLookup` was passing a bare IP to `net.Dialer` which requires `host:port`. Fixed with `net.JoinHostPort(gateway, "53")`.

### Build
- **Multi-stage Dockerfile** — rewrote Dockerfile to build from source (node/pnpm frontend → Go backend → alpine runtime) instead of downloading a pre-built binary.
- **Go BuildKit cache mount** — added `--mount=type=cache` for `/root/.cache/go-build` and `/go/pkg/mod` so incremental Go builds take minutes instead of 30+.

### Per-Client DNS Blocking Profiles
Full feature allowing different blocking rules per client or subnet.

**Backend:**
- New DB models: `Profile`, `ProfileSource`, `ProfileCustomBlacklist`, `ProfileWhitelist`, `SubnetProfile`
- `backend/profile/` package — service, repository, and model layer
- Per-profile in-memory caches (block + whitelist) rebuilt on change
- Subnet matching — CIDR rules sorted most-specific first, checked before Default fallback
- DNS handler routes non-Default clients through profile caches at query time
- Default profile preserves existing behaviour with zero regression
- `profile_sources` synced to all profiles when a new blocklist is added
- Client assignment creates a stub `mac_addresses` row when MAC wasn't resolved via ARP (normal inside Docker)

**REST API** (`/api/profiles`, `/api/subnets`, `/api/client/:ip/profile/:profileId`):
- Full CRUD for profiles (create, rename, delete)
- Per-profile source toggle (enable/disable a blocklist for one profile)
- Per-profile custom blacklist and whitelist (add/remove domains)
- Subnet → profile assignment (CIDR ranges)
- Client IP → profile assignment

**Frontend (`/profiles` page):**
- Profile cards with source count
- Create / delete profiles
- Side sheet per profile with four tabs:
  - **Clients** — assign/remove individual IPs
  - **Sources** — toggle each blocklist on/off per profile
  - **Blacklist** — custom blocked domains for this profile
  - **Whitelist** — custom allowed domains for this profile
- Subnet rules panel — add/delete CIDR → profile mappings with profile name shown
- Sidebar nav item added

---

## Known Issues / In Progress
- **Bypass toggle** on the Clients page has the same silent-failure bug for clients without a detected MAC (existing upstream issue, outside profile scope)
- Sources tab showed empty on first open — fixed by fetching full profile detail (`GET /profiles/:id`) when the sheet opens rather than relying on the list response

---

## To Do

### Short Term
- [ ] Verify subnet rules show correct profile name after `c_id_r` column fix
- [ ] Verify client IP assignment works for OCTANE (stub row creation)
- [ ] Verify sources populate correctly in the Sources tab after full-profile fetch fix
- [ ] Test that blocking actually works per-profile (query from Wife IP uses Wife's source set)

### Nice to Have
- [ ] Show assigned profile name on each client card / client detail page
- [ ] Allow assigning a profile directly from the existing Clients network map page
- [ ] Profile stats on the card (custom blocked domains count, whitelisted count)
- [ ] Confirm / warn before deleting a profile that has clients assigned to it
- [ ] Rename the `c_id_r` column to `cidr` properly (requires SQLite migration)

### Upstream PR (separate branch)
- [ ] Open PR to pommee/goaway for the two DNS fixes (SERVFAIL + client hostname)
  - Already isolated on `main` branch of fork — just needs a PR opened
