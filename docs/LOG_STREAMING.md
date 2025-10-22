# Job Log Streaming (Beta Demonstration)

> Last Updated: 2025-10-17
> Status: Active

This document explains the experimental Server‑Sent Events (SSE) based job log streaming feature that lets you observe the output of executed example jobs in near real time.

## Endpoint

```
GET /api/v1/beta/examples/run/:job_id/logs
```

Returns an SSE stream with the following event types (all `data:` lines are UTF‑8 text):

| Event  | Meaning | Notes |
|--------|---------|-------|
| `status` | Initial job state snapshot | `data: state=<queued|running|done|failed|timeout>` |
| `log`    | One chunk of job output    | Current implementation sends the full captured output once; future versions may stream line by line. |
| `done`   | Stream finished            | `data: END` if complete output was sent, or `data: PENDING` if the job had no output yet (still running or queued). |

## Current Behavior (MVP)

1. On connect the server emits a `status` event.
2. If the job has not produced output yet and is still `queued` or `running`, a placeholder `log` event is sent followed by `done` with `PENDING`.
3. If output is already captured, the server emits a single `log` event containing the entire output, then a `done` event with `END`.
4. The connection closes after `done`.

This is intentionally simple for beta clarity. A production implementation would:

- Stream incremental lines (tail) as they are produced.
- Keep the connection open until job completion.
- Support client cancellation and back‑pressure handling.
- Potentially multiplex multiple event types (stderr vs stdout, progress updates, heartbeats).

## Frontend Usage

The main page (`web/templates/index.html`) contains an experimental panel:

```js
const es = new EventSource(`/api/v1/beta/examples/run/${jobId}/logs`);
es.addEventListener('status', ev => console.log('status', ev.data));
es.addEventListener('log', ev => console.log('log', ev.data));
es.addEventListener('done', ev => es.close());
```

## Example cURL Session

```
# Run an example (returns a job id in JSON)
curl -s -X POST http://localhost:8080/api/v1/beta/examples/run -d '{"example_id":"some_example"}' -H 'Content-Type: application/json'

# Stream logs (replace JOB_ID)
curl -N http://localhost:8080/api/v1/beta/examples/run/JOB_ID/logs
```

You should see something like:

```
event: status
data: state=done

event: log
data: (output not yet available - job still running)

event: done
data: PENDING
```

(or the actual captured output if already finished).

## Future Enhancements (Suggested)

- Real‑time incremental line streaming (buffered channel per job).
- Reconnect & resume (Last-Event-ID) support.
- Filtering event types (query param `?events=log,status`).
- Heartbeat events to keep long‑running connections alive.
- Authorization / rate limiting per client.

## Beta Notes

This feature is **NOT** hardened for production: it lacks authentication, per‑client quotas, reconnection logic, proper error classification, and efficient log buffering. It is intentionally minimal to illustrate the mechanics of SSE and how an async job model can expose progressive output.

---
*Beta Demonstration – For learning purposes only.*

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
