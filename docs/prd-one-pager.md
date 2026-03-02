# Kanban CLI — Product Requirements Document

**Version:** 0.1  
**Date:** 2026-03-01  
**Status:** Draft

---

## Overview

A terminal-based Kanban board for developers who live in the CLI. Tasks are persisted in a local SQLite database. The UI is built with Bubbletea and renders entirely in the terminal.

---

## Problem

Existing Kanban tools (Linear, Trello, Jira) require a browser or native app. Developers who prefer the terminal have no lightweight, keyboard-driven option that stays local and requires zero setup.

---

## Goals

- Full Kanban workflow without leaving the terminal
- Zero external dependencies for the end user (single binary)
- Data stored locally in SQLite — no accounts, no sync, no internet
- Columns and priorities are user-configurable

## Non-Goals

- Multi-user collaboration or sync
- Cloud storage or remote backends
- Mobile or web interface
- Import/export to third-party tools (v1)

---

## Users

Solo developers and power users who prefer keyboard-driven, terminal-native tooling.

---

## Core Features

| Feature | Description |
|---|---|
| Configurable columns | User creates, renames, reorders, and colors columns |
| Task management | Create, edit, move, and delete tasks |
| Priority levels | Low, Medium, High per task |
| Default task titles | Auto-generated as "Task {id}" if no title provided |
| Keyboard navigation | Full keyboard control — no mouse required |
| Persistent storage | SQLite database stored locally |

---

## Success Metrics

- End-to-end task creation to completion in under 10 keystrokes
- App launches in under 200ms
- Zero data loss on unexpected exit
