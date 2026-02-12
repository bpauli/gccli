# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-12

Initial release of gccli — a fast, script-friendly CLI for Garmin Connect.

### Added

- **Authentication** — Browser SSO and headless email/password login with MFA support, OAuth1-to-OAuth2 token exchange
- **Secure credential storage** — OS keyring (macOS Keychain, Linux Secret Service, encrypted file fallback), tokens keyed by `gccli:token:<email>`
- **Activities** — List, search, count, view details/splits/weather/HR zones/power zones/exercise sets/gear, download (FIT/GPX/TCX/KML/CSV), upload, create manual entries, rename, retype, delete
- **Health data** — Daily summaries, steps (daily/weekly), heart rate, resting HR, floors, sleep, stress (daily/weekly), respiration, SpO2, HRV, body battery, training readiness/status, fitness age, VO2max/max metrics, lactate threshold, cycling FTP, race predictions, endurance score, hill score, intensity minutes, wellness events
- **Body composition** — Weight tracking, body fat, muscle mass, weigh-in history, blood pressure, FIT file encoding for composition uploads
- **Workouts** — List, view details, download as FIT, upload from JSON, create with pace/HR/power/cadence targets, schedule, delete
- **Devices** — List registered devices, view settings, solar data, alarms, primary/last-used device
- **Gear** — List gear, usage stats, linked activities, defaults per activity type, link/unlink to activities
- **Goals & badges** — Active goals, earned/available/in-progress badges, challenges, personal records
- **Training plans** — Browse and view training plans
- **Profile** — User profile and settings
- **Hydration** — View and log daily water intake
- **Wellness** — Menstrual cycle data, pregnancy summary
- **Output formats** — JSON (`--json`), plain/TSV (`--plain`), and human-friendly tables (default)
- **Multiple accounts** — `--account` flag and `GCCLI_ACCOUNT` env var
- **Resilient networking** — Automatic retry on 429/5xx with exponential backoff, circuit breaker for fault tolerance, auto-refresh OAuth2 tokens on 401
- **CI pipeline** — GitHub Actions for fmt-check, lint, and test
- **Cross-platform builds** — macOS (amd64/arm64) and Linux (amd64/arm64) via goreleaser

[0.1.0]: https://github.com/bpauli/gccli/releases/tag/v0.1.0
