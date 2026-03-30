# NYX Tool Matrix

This document records the NYX-native operator tool surface and the worker-image
capabilities that support it.

## Operator-Visible Functions

| Category | Capability | NYX surface | Implementation |
| --- | --- | --- | --- |
| Barrier | Complete or escalate work | `done`, `ask` | Runtime workflow controls |
| Environment | Terminal execution | `terminal_exec` | Function registry + executor manager |
| Environment | File operations | `file_read`, `file_write` | Function registry + workspace validation |
| Browser | Inspect target content | `browser`, `browser_html`, `browser_markdown`, `browser_links`, `browser_screenshot` | Browser service |
| Search | Public web search | `search_web`, `search_deep`, `search_exploits` | Search service providers |
| Memory | Store and retrieve context | `store_memory`, `search_memory` | Memory service + repository |
| Reporting | Export evidence | flow report endpoints | Reports package |

## Worker Image Capabilities

| Capability group | Current NYX support |
| --- | --- |
| General shell work | `NYX_EXECUTOR_IMAGE` |
| Security tooling | `NYX_EXECUTOR_IMAGE_FOR_PENTEST` |
| Network mode control | `NYX_EXECUTOR_NETWORK_MODE` and `NYX_EXECUTOR_NETWORK_NAME` |
| Optional raw sockets | `NYX_EXECUTOR_ENABLE_NET_RAW` |
| Browser automation | Chromium + `chromedp` |
| Search providers | DuckDuckGo, SearxNG, Tavily, Perplexity, Sploitus |

## Notes

- NYX favors clear operator-visible capabilities over large undocumented tool catalogs.
- Image selection, network mode, and execution metadata are part of the runtime contract.
- Capability growth should land in NYX-owned modules rather than external reference repos.
