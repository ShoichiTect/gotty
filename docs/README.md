# GoTTY Fork — Documentation

## Structure

```
docs/
├── README.md              ← this file
├── guides/                ← permanent references (read first)
├── plans/                 ← strategy, roadmaps, backlogs
├── investigations/        ← audit reports, research findings
└── completed/             ← features already implemented
```

---

## 📘 Guides — read first

| File | Description |
|------|-------------|
| [domain-primer.md](guides/domain-primer.md) | Architecture overview, terminology (PTY, Master/Slave, WebTTY protocol), code map |
| [gotty-ipad-setup.md](guides/gotty-ipad-setup.md) | Build, deploy, and access GoTTY from iPad via Tailscale |

---

## 🎯 Plans & strategy

| File | Status | Description |
|------|--------|-------------|
| [maintenance-plan.md](plans/maintenance-plan.md) | **Ongoing** | Overall roadmap: Phase 0–6 (Phase 0–2.5 ✅ done) |
| [hcl-to-json-migration-plan.md](plans/hcl-to-json-migration-plan.md) | ✅ Done (2da9307) | Migration from HCL config to JSON |
| [outdated-patterns.md](plans/outdated-patterns.md) | Backlog | Catalog of 17 legacy code patterns to modernize |

---

## 📋 Investigations & audits

| File | Description |
|------|-------------|
| [dependency-security-audit.md](investigations/dependency-security-audit.md) | Dependency audit, npm/Go vulnerability scan results, security design findings |
| [ipad-ttyd-investigation.md](investigations/ipad-ttyd-investigation.md) | Root-cause analysis: why ttyd fails on iPad while GoTTY works |
| [font-size-investigation.md](investigations/font-size-investigation.md) | Can font size be changed via CLI? Options and code patches |
| [ipad-paste-issue.md](investigations/ipad-paste-issue.md) | iPad Safari paste issue: clipboard content not pasted into terminal |

---

## ✅ Completed features

| File | Description |
|------|-------------|
| [diagnostics.md](completed/diagnostics.md) | `--debug` flag: WebSocket/HTTP/proxy header logging |
| [ipad-ctrl-c-fix.md](completed/ipad-ctrl-c-fix.md) | iPadOS Safari Ctrl+C → SIGINT fix (keyboard event intercept) |
