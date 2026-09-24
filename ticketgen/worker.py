"""Poll Postgres for queued generation jobs and run the atomic AI pipeline."""

from __future__ import annotations

import json
import logging
import os
import socket
import time
import uuid
from datetime import datetime, timezone

import psycopg

from catalog import load_catalog
from llm import LLMClient
from pipeline import Pipeline, PipelineState

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
log = logging.getLogger("ticketgen.worker")

LEASE_SECONDS = int(os.environ.get("TICKETGEN_LEASE_SECONDS", "120"))
POLL_SECONDS = float(os.environ.get("TICKETGEN_POLL_SECONDS", "2"))
WORKER_ID = os.environ.get("TICKETGEN_WORKER_ID") or f"{socket.gethostname()}-{uuid.uuid4().hex[:8]}"
MAX_JOB_ATTEMPTS = int(os.environ.get("TICKETGEN_MAX_JOB_ATTEMPTS", "3"))

IN_PROGRESS = (
    "enriching",
    "filling_pii",
    "picking_type",
    "tagging_type",
    "tagging_common",
    "building_services",
    "building_ref",
)

_IN_PROGRESS_SQL = ", ".join(["%s"] * len(IN_PROGRESS))

CLAIM_SQL = f"""
UPDATE ticket_generation_jobs AS j SET
    status = 'enriching',
    version = j.version + 1,
    claimed_by = %s,
    claimed_at = now(),
    lease_until = now() + (%s || ' seconds')::interval,
    updated_at = now(),
    attempts = j.attempts + 1,
    error_message = ''
WHERE j.id = (
    SELECT id FROM ticket_generation_jobs
    WHERE status = 'queued'
       OR (
            status IN ({_IN_PROGRESS_SQL})
            AND lease_until IS NOT NULL
            AND lease_until < now()
       )
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING
    id, group_id, prompt, status, version, scenario_text, draft_title,
    draft_reference, attempts
"""

CAS_SQL = """
UPDATE ticket_generation_jobs SET
    status = %s,
    version = version + 1,
    scenario_text = %s,
    draft_title = %s,
    draft_reference = %s::jsonb,
    error_message = %s,
    claimed_by = %s,
    claimed_at = %s,
    lease_until = %s,
    updated_at = now()
WHERE id = %s AND status = %s AND version = %s
"""


def utcnow() -> datetime:
    return datetime.now(timezone.utc)


def claim_job(conn: psycopg.Connection) -> dict | None:
    with conn.cursor() as cur:
        cur.execute(CLAIM_SQL, (WORKER_ID, str(LEASE_SECONDS), *IN_PROGRESS))
        row = cur.fetchone()
        if not row:
            return None
        cols = [d.name for d in cur.description]
        return dict(zip(cols, row))


def cas_update(
    conn: psycopg.Connection,
    *,
    job_id,
    expected_status: str,
    expected_version: int,
    status: str,
    scenario_text: str,
    draft_title: str,
    draft_reference: dict,
    error_message: str = "",
    clear_claim: bool = False,
    refresh_lease: bool = True,
) -> bool:
    claimed_by = "" if clear_claim else WORKER_ID
    claimed_at = None if clear_claim else utcnow()
    if clear_claim:
        lease_until = None
    elif refresh_lease:
        from datetime import timedelta

        lease_until = utcnow() + timedelta(seconds=LEASE_SECONDS)
    else:
        lease_until = None
    with conn.cursor() as cur:
        cur.execute(
            CAS_SQL,
            (
                status,
                scenario_text,
                draft_title,
                json.dumps(draft_reference, ensure_ascii=False),
                error_message,
                claimed_by,
                claimed_at,
                lease_until,
                job_id,
                expected_status,
                expected_version,
            ),
        )
        return cur.rowcount == 1


def process_job(conn: psycopg.Connection, pipeline: Pipeline, job: dict) -> None:
    job_id = job["id"]
    version = int(job["version"])
    status = job["status"]
    prompt = job["prompt"]
    log.info("processing job=%s attempt=%s", job_id, job["attempts"])

    state = PipelineState(prompt=prompt)
    draft = job.get("draft_reference") or {}
    if isinstance(draft, str):
        draft = json.loads(draft)

    def save(next_status: str, *, clear_claim: bool = False) -> bool:
        nonlocal version, status
        ok = cas_update(
            conn,
            job_id=job_id,
            expected_status=status,
            expected_version=version,
            status=next_status,
            scenario_text=state.scenario,
            draft_title=state.title,
            draft_reference=state.draft_reference(),
            clear_claim=clear_claim,
        )
        if not ok:
            return False
        version += 1
        status = next_status
        return True

    try:
        state.title, state.scenario = pipeline.generate_scenario(prompt)
        if not save("filling_pii"):
            log.warning("lost claim after scenario job=%s", job_id)
            return

        state.pii = pipeline.fill_pii()
        if state.pii.get("address") and state.pii["address"] not in state.scenario:
            state.scenario = f"{state.scenario} Адрес: {state.pii['address']}."
        if not save("picking_type"):
            log.warning("lost claim after pii job=%s", job_id)
            return

        state.type_code = pipeline.pick_type(state.scenario, prompt)
        if not save("tagging_type"):
            log.warning("lost claim after type job=%s", job_id)
            return

        state.type_tags = pipeline.fill_type_tags(state.scenario, state.type_code)
        if not save("tagging_common"):
            log.warning("lost claim after type tags job=%s", job_id)
            return

        state.common_tags = pipeline.fill_common_tags(state.scenario, state.type_code)
        if not save("building_services"):
            log.warning("lost claim after common tags job=%s", job_id)
            return

        state.service_codes = pipeline.pick_services(
            state.scenario, state.type_code, state.all_tags()
        )
        ref = state.draft_reference()
        pipeline.catalog.validate_reference(ref)

        if not save("ready", clear_claim=True):
            log.warning("lost claim on ready job=%s", job_id)
            return
        log.info(
            "job=%s ready title=%r type=%s tags=%d",
            job_id,
            state.title,
            state.type_code,
            len(state.all_tags()),
        )
    except Exception as e:  # noqa: BLE001
        log.exception("job=%s failed", job_id)
        err = str(e)[:1000]
        cas_update(
            conn,
            job_id=job_id,
            expected_status=status,
            expected_version=version,
            status="failed",
            scenario_text=state.scenario or (job.get("scenario_text") or ""),
            draft_title=state.title or (job.get("draft_title") or ""),
            draft_reference=state.draft_reference() if state.type_code or state.pii else draft,
            error_message=err,
            clear_claim=True,
        )


def main() -> None:
    dsn = os.environ.get("TICKETGEN_POSTGRES_DSN") or os.environ.get("TRAINEEBOX_POSTGRES_DSN")
    if not dsn:
        raise SystemExit("TICKETGEN_POSTGRES_DSN or TRAINEEBOX_POSTGRES_DSN required")
    catalog_path = os.environ.get("TICKETGEN_CATALOG_PATH") or os.environ.get("TRAINEEBOX_CATALOG_PATH")
    if not catalog_path:
        raise SystemExit("TICKETGEN_CATALOG_PATH or TRAINEEBOX_CATALOG_PATH required")

    catalog = load_catalog(catalog_path)
    pipeline = Pipeline(catalog, LLMClient())
    backends = getattr(pipeline.llm, "backends", [])
    log.info(
        "worker=%s catalog_types=%d backends=%d",
        WORKER_ID,
        len(catalog.types),
        len(backends),
    )

    while True:
        try:
            with psycopg.connect(dsn, autocommit=True) as conn:
                job = claim_job(conn)
                if job is None:
                    time.sleep(POLL_SECONDS)
                    continue
                process_job(conn, pipeline, job)
        except Exception:  # noqa: BLE001
            log.exception("worker loop error")
            time.sleep(POLL_SECONDS)


if __name__ == "__main__":
    main()
