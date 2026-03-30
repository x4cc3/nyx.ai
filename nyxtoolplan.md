# NYX Tooling Roadmap

## Goal

Expand NYX from a narrow function gateway into a production-grade security
operations tool platform with isolated workers, richer search capability,
clear execution metadata, and strong operator controls.

## Current Baseline

NYX already provides:

- terminal execution through local or Docker executor modes
- file reads and writes inside isolated workspaces
- browser automation via `chromedp`
- public web search through DuckDuckGo or SearxNG
- semantic memory backed by pgvector or deterministic fallback embeddings
- report generation in Markdown, JSON, and PDF

## Priority Tracks

### 1. Worker Images And Execution Policy

- keep a general worker image for low-risk tasks
- keep a pentest worker image for authorized security tooling
- make network mode, image choice, and raw-socket use visible in metadata
- ensure Docker-mode startup fails closed when prerequisites are missing

### 2. Tool Surface Growth

- preserve the existing terminal, file, browser, search, and report surfaces
- add higher-signal wrappers only when they improve operator clarity
- keep tool interfaces scoped, typed, and easy to audit

### 3. Search And Memory

- maintain more than one public search provider
- keep exploit-oriented research available through NYX-native search functions
- separate stored memory by purpose, such as observations, notes, and references

### 4. Evidence And Reporting

- persist stdout, stderr, exit code, duration, image, and network mode
- keep report exports consistent across API and UI
- make findings and artifacts easy to inspect after execution

### 5. Operator Safety

- keep scope validation mandatory for browser and terminal work
- require approvals for high-risk behavior where policy demands it
- keep rate limits, body limits, and audit logging enabled by default

## Validation Standard

NYX is in a good state when:

- the Go services build and test cleanly
- the frontend builds cleanly
- Docker images build successfully
- the production compose stack starts with healthy services
- operators can understand what ran, where it ran, and what it produced
