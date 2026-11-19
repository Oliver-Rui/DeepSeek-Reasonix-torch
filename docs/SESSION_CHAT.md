---
owner: @SivanCola
backup: @esengine
status: active
reviewed: 2026-11-19
---

# Session chat (会话互聊)

The feature this fork rebuilds is MiMo Desktop's 会话互聊, taken apart from
its shipped Electron bundle (`runtime-res/releases/renderer/assets/*.js`,
`app.asar`). This document is the acceptance list: the shipped layout, wording
and behaviour, and where this tree already differs.

The fork's name for the whole loop is **会话互聊**. The previous name in this
tree, 会话协作, is retired from user-visible strings.

## What the shipped feature is

- An experimental switch in Settings → 实验室功能 → 会话互聊.
- When on, every session is a contact. The model gets eight tools: `contacts`,
  `talk_to_session`, `create_session`, `update_session`,
  `list_external_sessions`, `read_external_session`, `report_event`,
  `event_ledger`.
- A session can call another session and wait for its real turn (sync), or hand
  a task over and receive the reply later (async).
- Local external agents (Claude Code, Codex, Gemini CLI, Cursor) are contacts
  too, reached over MCP through a loopback gateway.
- One session may be the 大管家 (default session): reported events go there.

## Surfaces

### 0. What a user actually sees

The roster below is **not** part of session chat. It is a separate internal
feature gated twice: its own lab switch, and a server answer that is true only
on the company intranet (`sessionManagerAvailable`, with `available: false` as
the initial state). A public install shows none of it. What a person does see:

1. The sidebar row's hover card and its chat-link chips.
2. The timeline's 来自「X」 label on a relayed message.
3. The lab switch and the agent-gateway block in settings.

This tree builds those three. The roster stays documented below as the shape
the shipped product uses when its gate is open, and is not built here.

### 1. Roster page (会话) — gated, not built

A full page, not a popover. Its own heading and subtitle:

- heading 会话, subtitle 像团队名册一样管理登记了用途的会话：谁负责什么、在干什么、和谁协作。
- 新建会话 button at the right of the heading row.
- Manager card: 大管家, 等你定 N 件, 进行中 N 件, 选择/更换.
- Filter row: status tabs 全部 / 进行中 / 阻塞 / 完成 / 空闲 with counts, a project
  select (全部项目), and a 220px search box (搜索标题、用途或当前任务).
- Groups: 需要关注 (blocked), 进行中 (working), 其他. Each group header is a 3px
  coloured bar, the name, and the count. Empty groups show their own line
  (没有阻塞的会话 / 现在没人在干活 / 没有).
- Cards are a responsive grid: one column, three columns at ≥720px container
  width, 14px gap. Each card is 12px radius, 14px/12px padding, 1px border.
  A blocked card carries a 3px warning bar down its left edge.
- Card body, first row: 36px avatar, title, purpose, status pill.
  - The avatar takes its colour from the id and shows the initial letter.
  - The title is 14px medium and truncates; a pulsing 8px green dot marks busy.
  - The purpose is 12px muted, one line.
- Card body, second row: 当前：<task> at 13px.
- Card body, then: one row per chat link (`↗ 等待「X」回信` plus a 打开 button).
- Card footer, above a border: 手上 N 件, project name, relative time.
- External agents are a separate section at the bottom (外部 agent), wrapping
  chips.
- Clicking a card opens a fixed detail panel.
  - It is a 380px aside anchored to the window: top 48px, right and bottom
    12px, 14px radius, float shadow.
  - Header: 44px avatar, title at 16px, a status pill and busy dot, a 关闭
    button, and the purpose.
  - Then an 打开对话 button with a ⋯ menu: 设为大管家, 置顶, 归档, 删除.
  - Then the scrollable sections 手上的事 and 协作往来.
- Status pills: 进行中 blue, 阻塞 warning, 完成 green, 空闲/未设状态 neutral.
- Event statuses: 新事件, 排队, 处理中, 等你定, 完成, 失败, 忽略.

### 2. Sidebar hover card

Hovering a session row shows a small card (max 360px wide, min 224px):

- title (medium, wraps anywhere)
- 项目 <name>
- 更新于 <relative time>
- running subagents, joined by 、
- one chat-link line per link: 11px, blue, a dot (outgoing: ringed pulse;
  incoming: plain dot) and the text:
  - outbound → 等待「X」回信
  - inbound request → 处理「X」的消息
  - inbound reply → 处理「X」的回信
  - queued suffix → ` · 排队中（第 N 位）`
  - external peers read `X(外部 agent)`
  - clicking navigates to the peer's session (title 点击查看对方会话)

Ordering: queued links last, then by queue position.

**Position.** The card opens beside the row it belongs to and stays there: its
top edge level with the row's top, its left edge just past the sidebar. This is
a rule, not a preference — a card that drifts away from its row is broken even
when its contents are right. Two things make it drift, and both are load-bearing:

| Cause | Rule |
|---|---|
| The window scales itself with `zoom` on the root element | Divide the anchor's offset by the live `zoom` before setting `top`/`left`, as every fixed surface in this tree does |
| `.app` carries `overflow: clip`, which makes it the containing block for a fixed child | Mount the card on the document body, never inside the sidebar |

### 3. Timeline source label

A message the model received from another session carries a label above the
bubble, aligned right: a 14px chat icon and 来自「X」 / 「X」的回信 (or 来自外部
agent「X」). It is a button that opens the source session; when the source is
gone the label is inert and reads 来源会话已删除或当前不可用.

### 4. Settings

实验室功能 → 会话互聊: one switch, described as experimental, changes apply to
new turns, restart to apply. Under it, the agent gateway block:

- 让其他本地 agent 接入 (loopback only)
- 默认会话 (大管家): unset means such calls and events are rejected
- 一键接入: writes MiMo Desktop into Claude Code / Codex / Gemini CLI /
  Cursor MCP config, previewing the exact change; never overwrites an existing
  entry of the same name
- 已连接的外部 agent (per-agent disable)
- 调用记录 (last 7 days: time, caller, callee, mode, result — no content)

## Message protocol

- Request header: `【会话对话|来自会话「X」】` followed by the body.
- Reply header: `【会话对话|会话「X」的回信】`.
- External request/reply use `来自外部 agent「X」`.
- Event notification: `[event <id>] source=<s> type=<t> subject=<sub>` then
  `summary:` and optional `link:`.
- The model sees the header; the UI reads structured metadata (session id,
  title, direction), never the text.

## Timing and limits

| Rule | Shipped value |
|---|---|
| Sync wait | 120s, then reported as busy |
| Async admission confirm | 15s |
| External agent call | 180s hard timeout |
| Chain depth | 50 hops |
| Event merge window | 10 minutes |
| Inbound message | 64 KiB |
| External read | 60 KB, tool content hidden by default |
| External list | 20 default, 100 max |
| Gateway rate | 60 requests/min/principal |

## What this tree has today

- The hover card, its chat-link chips, and the timeline's 来自「X」 label.
- A bottom-rail dispatch board, peers, dispatch, and a rename tool.
- No roster page, no agent gateway, no event ledger.
- User-visible strings say 会话互聊.

## Acceptance checks

Each of these was a real break found by driving a dispatch through the running
app, and each is pinned by a test. They are the things to try again after any
change to the delivery path, the board, or the card.

| # | Check | What a break looks like |
|---|---|---|
| 1 | Dispatch two tasks from one session; both targets answer | The leader waits forever, or a target's turn never starts |
| 2 | The reply lands on the leader as a new turn | It sits `queued` in the inbox while the leader's turn runs |
| 3 | The reply's message carries `peer` in `/history` | `peer` is null and the timeline shows no label |
| 4 | A reply reads 「X」的回信; a first handover reads 来自「X」 | Everything reads 来自「X」, or `reply` is always absent |
| 5 | The card names the session, not its file name | The label reads `20261119-...-dispatch` |
| 6 | The card sits level with the row it describes, before and after scrolling the sidebar | It drifts down the page, or away when the rail scrolls |
| 7 | The board lists both ends with role, project and state | A row is missing, or a name is blank |
| 8 | The board appears without a reload after the page was in the background | It stays empty until the window is reloaded |
| 9 | Archiving or deleting an end removes its board row and its card link | A closed chapter keeps asking for attention |
| 10 | A leader that waits with a sleep still receives the answer | It burns its turn waiting and reads the reply late |
| 11 | Neither dispatched session asks the person to approve a tool; only the session the person opened does | A crew member sits mid-task on a card nobody is looking at |

Checks 2, 6, 8, 10 and 11 are the ones a unit test cannot fully stand in for:
they depend on a real turn running, a real window being hidden or scrolled, or a
real pane being claimed by a dispatch.

## An unattended session

A session another session opened for a task has nobody at its window. The person
is looking at the pane that dispatched it, and the dispatcher cannot answer a
card either — so a card there is a stall, not a question. Three rules follow:

| Rule | Why |
|---|---|
| The grant hands `ask` on as `auto` | The lead cannot answer for a pane it is not looking at |
| Auto covers shapes static analysis cannot read (nested shells, interpreter `-c`) | The same reason the headless path allows them: refusing unattended only buys a round trip |
| A rule that still wants a person refuses with its reason | The model does the rest and reports what it could not do, instead of waiting |

The gate is built when a pane opens, before anyone knows a dispatch will claim
it, so claiming it must rebuild the gate. A pane that keeps the gate it was born
with keeps the posture of a session somebody is watching, and stalls.

## Rebuild order

1. Rename user-visible strings to 会话互聊 — done.
2. Sidebar hover card and chat-link chips — done.
3. The timeline's 来自「X」 label, carried as `provider.PeerSource` — done.
4. Settings: the lab switch, then the gateway block.
5. Tools: the eight names and their schemas.
6. Event ledger and the 大管家 routing.

The roster is out of scope: its gate is the company intranet, so building it
would add a surface the shipped product does not show a public install.

Each step lands on `area/collab` (or `torch` when it also touches the kernel),
per `docs/BRANCHES.md`.
