# bs-coding Eval Files

## Files

| File | Total | Good | Bad |
|---|---|---|---|
| `evals/brightsign-player-evals.json` | 250 | 125 | 125 |
| `evals/gopurple-evals.json` | 250 | 125 | 125 |

## BrightSign Player Evals

Questions about coding for the BrightSign player, derived from the `technical-documentation/` corpus. Covers 24 source files across:

- BrightScript language fundamentals (syntax, types, variables, control flow, objects)
- Practical development (SD card deployment, autorun.zip, DWS, SFTP)
- JavaScript/HTML5 playback and Node.js programs
- Debugging (BrightScript telnet/SSH, Chrome DevTools)
- Design patterns and plugin architecture
- Hardware integrations (GPIO, serial, USB, touch)
- NPU/AI programming with native C/C++ extensions
- BSN.cloud integration, automated provisioning, per-player control
- How-to guides (REST APIs, multi-zone layouts, DHCP server, etc.)

Difficulty mix: ~40% beginner, ~35% intermediate, ~25% advanced.

## GoPurple Cloud API Evals

Questions about coding against BSN.cloud using the GoPurple Go SDK, derived from the `gopurple/` corpus. Covers 13 source files across:

- SDK initialization, authentication, and configuration
- Device listing, status, errors, and group management
- RDWS operations (reboot, screenshots, file ops, network diagnostics, registry, logs, SSH/telnet)
- B-Deploy setup CRUD and the full 80+ field configuration reference
- B-Deploy device association and provisioning workflows
- Subscription management
- Type safety and exported type patterns
- Error handling
- Raw API scripts and endpoint coverage

Difficulty mix: ~40% beginner, ~35% intermediate, ~25% advanced.

## Bad Eval Strategy

All `bad` evals use real BrightSign/GoPurple terminology combined with fabricated but plausible-sounding methods, objects, types, configuration fields, or API endpoints. They are designed to be indistinguishable from real questions without actually checking the corpus.
