#!/usr/bin/env python3
# /// script
# dependencies = [
#     "httpx2",
#     "rich",
# ]
# ///
"""
test_hoverfly.py — Run every curl command from data/curl_dump.txt against
a running Hoverfly instance and report pass/fail with rich formatting.

Usage:
    uv run test_hoverfly.py [--url URL] [--dump FILE] [--fail-only] [--json]

Options:
    --url       Base URL of the Hoverfly instance (default: http://localhost:8090)
    --dump      Path to the curl dump file        (default: data/curl_dump.txt)
    --fail-only Only print failing tests
    --json      Write results to hoverfly_test_results.json
    --timeout   Per-request timeout in seconds    (default: 10)
    --parallel  Number of parallel workers         (default: 1, sequential)
"""

import argparse
import json
import re
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pathlib import Path
from typing import Optional

import httpx2
from rich import box
from rich.console import Console
from rich.panel import Panel
from rich.progress import (
    BarColumn,
    MofNCompleteColumn,
    Progress,
    SpinnerColumn,
    TaskProgressColumn,
    TextColumn,
    TimeElapsedColumn,
)
from rich.table import Table
from rich.text import Text

console = Console()

# ──────────────────────────────────────────────────────────────────────────────
# Data model
# ──────────────────────────────────────────────────────────────────────────────


@dataclass
class TestCase:
    index: int
    method: str
    payload: dict
    raw: str


@dataclass
class TestResult:
    case: TestCase
    status: str  # "pass" | "fail" | "error"
    http_code: int = 0
    response: Optional[dict] = None
    error_msg: str = ""
    elapsed_ms: float = 0.0
    detail: str = ""


# ──────────────────────────────────────────────────────────────────────────────
# Parser — extract individual curl commands from the dump file
# ──────────────────────────────────────────────────────────────────────────────

_CURL_RE = re.compile(r"curl\s+.*?http://localhost:\d+", re.DOTALL)
_DATA_RE = re.compile(r"--data\s+'([^']+)'")


def load_test_cases(dump_path: Path) -> list[TestCase]:
    """Parse the curl dump file into a list of TestCase objects."""
    raw_text = dump_path.read_text()

    # Split on blank lines, then collect all curl commands
    cases: list[TestCase] = []
    idx = 0
    for line in raw_text.splitlines():
        line = line.strip()
        if not line.startswith("curl"):
            continue

        m = _DATA_RE.search(line)
        if not m:
            continue

        data_str = m.group(1)
        try:
            payload = json.loads(data_str)
        except json.JSONDecodeError:
            # Some examples in the dump have intentional JSON errors; skip.
            idx += 1
            cases.append(
                TestCase(
                    index=idx,
                    method=f"<malformed #{idx}>",
                    payload={},
                    raw=line,
                )
            )
            continue

        method = payload.get("method", "<unknown>")
        idx += 1
        cases.append(TestCase(index=idx, method=method, payload=payload, raw=line))

    return cases


# ──────────────────────────────────────────────────────────────────────────────
# Runner — send one request and classify it
# ──────────────────────────────────────────────────────────────────────────────


def run_case(
    case: TestCase, url: str, timeout: float, client: httpx2.Client
) -> TestResult:
    if not case.payload:
        return TestResult(
            case=case,
            status="error",
            error_msg="Malformed JSON in curl dump — skipped",
            detail="SKIP",
        )

    t0 = time.perf_counter()
    try:
        resp = client.post(url, json=case.payload, timeout=timeout)
        elapsed = (time.perf_counter() - t0) * 1000
    except httpx2.ConnectError:
        return TestResult(
            case=case,
            status="error",
            http_code=0,
            error_msg="Connection refused — is Hoverfly running?",
            elapsed_ms=(time.perf_counter() - t0) * 1000,
        )
    except httpx2.TimeoutException:
        return TestResult(
            case=case,
            status="error",
            http_code=0,
            error_msg=f"Timeout after {timeout}s",
            elapsed_ms=(time.perf_counter() - t0) * 1000,
        )
    except Exception as exc:  # noqa: BLE001
        return TestResult(
            case=case,
            status="error",
            http_code=0,
            error_msg=str(exc),
            elapsed_ms=(time.perf_counter() - t0) * 1000,
        )

    try:
        body = resp.json()
    except Exception:  # noqa: BLE001
        return TestResult(
            case=case,
            status="fail",
            http_code=resp.status_code,
            error_msg="Response is not valid JSON",
            elapsed_ms=elapsed,
        )

    # JSON-RPC error field present → failure
    if "error" in body and body["error"] is not None:
        err = body["error"]
        if isinstance(err, dict):
            msg = err.get("message", str(err))
        else:
            msg = str(err)
        return TestResult(
            case=case,
            status="fail",
            http_code=resp.status_code,
            response=body,
            error_msg=msg,
            elapsed_ms=elapsed,
        )

    # result must be present (even if null — some methods legitimately return null)
    if "result" not in body:
        return TestResult(
            case=case,
            status="fail",
            http_code=resp.status_code,
            response=body,
            error_msg="Response missing 'result' field",
            elapsed_ms=elapsed,
        )

    return TestResult(
        case=case,
        status="pass",
        http_code=resp.status_code,
        response=body,
        elapsed_ms=elapsed,
    )


# ──────────────────────────────────────────────────────────────────────────────
# Reporting
# ──────────────────────────────────────────────────────────────────────────────

PASS_STYLE = "bold green"
FAIL_STYLE = "bold red"
ERROR_STYLE = "bold yellow"
SKIP_STYLE = "dim"


def status_text(r: TestResult) -> Text:
    if r.status == "pass":
        return Text("✓ PASS", style=PASS_STYLE)
    if r.status == "fail":
        return Text("✗ FAIL", style=FAIL_STYLE)
    if r.detail == "SKIP":
        return Text("⊘ SKIP", style=SKIP_STYLE)
    return Text("⚠ ERR", style=ERROR_STYLE)


def build_summary_table(results: list[TestResult], fail_only: bool) -> Table:
    table = Table(
        title="🛸  Hoverfly API Test Results",
        box=box.ROUNDED,
        show_header=True,
        header_style="bold cyan",
        expand=True,
    )
    table.add_column("#", style="dim", width=5, justify="right")
    table.add_column("Method", style="white", no_wrap=False, ratio=4)
    table.add_column("Status", width=9, justify="center")
    table.add_column("HTTP", width=5, justify="center")
    table.add_column("ms", width=7, justify="right")
    table.add_column("Detail", ratio=3, no_wrap=False)

    for r in results:
        if fail_only and r.status == "pass":
            continue

        detail = r.error_msg or ""
        if r.status == "pass" and r.response:
            result_val = r.response.get("result")
            if result_val is None:
                detail = "[dim]null[/dim]"
            elif isinstance(result_val, (dict, list)):
                detail = f"[dim]{type(result_val).__name__}[/dim]"
            else:
                preview = str(result_val)[:60]
                detail = f"[dim]{preview}[/dim]"

        http_str = str(r.http_code) if r.http_code else "-"
        ms_str = f"{r.elapsed_ms:.0f}" if r.elapsed_ms else "-"

        table.add_row(
            str(r.case.index),
            r.case.method,
            status_text(r),
            http_str,
            ms_str,
            detail,
        )

    return table


def build_scoreboard(results: list[TestResult]) -> Panel:
    passed = sum(1 for r in results if r.status == "pass")
    failed = sum(1 for r in results if r.status == "fail")
    errors = sum(1 for r in results if r.status == "error" and r.detail != "SKIP")
    skipped = sum(1 for r in results if r.detail == "SKIP")
    total = len(results)
    pct = 100 * passed / total if total else 0

    score_table = Table.grid(padding=(0, 2))
    score_table.add_column(justify="center")
    score_table.add_column(justify="center")
    score_table.add_column(justify="center")
    score_table.add_column(justify="center")
    score_table.add_column(justify="center")
    score_table.add_row(
        Text(f"✓ {passed}", style=PASS_STYLE),
        Text(f"✗ {failed}", style=FAIL_STYLE),
        Text(f"⚠ {errors}", style=ERROR_STYLE),
        Text(f"⊘ {skipped}", style=SKIP_STYLE),
        Text(f"{pct:.1f}% pass", style="bold white"),
    )
    score_table.add_row(
        Text("passed", style="dim"),
        Text("failed", style="dim"),
        Text("errors", style="dim"),
        Text("skipped", style="dim"),
        Text(f"of {total} total", style="dim"),
    )

    color = "green" if pct == 100 else ("yellow" if pct >= 80 else "red")
    return Panel(score_table, title="📊 Score", border_style=color, expand=False)


def build_failures_panel(results: list[TestResult]) -> Optional[Panel]:
    failures = [
        r for r in results if r.status in ("fail", "error") and r.detail != "SKIP"
    ]
    if not failures:
        return None

    content = Table.grid(padding=(0, 1))
    content.add_column(style="dim", width=4, justify="right")
    content.add_column(style="yellow")
    content.add_column()

    for r in failures:
        style = FAIL_STYLE if r.status == "fail" else ERROR_STYLE
        content.add_row(
            str(r.case.index),
            Text(r.case.method, style=style),
            Text(r.error_msg, style="dim"),
        )

    return Panel(
        content, title=f"❌ Failures & Errors ({len(failures)})", border_style="red"
    )


# ──────────────────────────────────────────────────────────────────────────────
# Entry point
# ──────────────────────────────────────────────────────────────────────────────


def main() -> None:
    parser = argparse.ArgumentParser(description="Hoverfly API smoke-test runner")
    parser.add_argument(
        "--url", default="http://localhost:8090", help="Hoverfly base URL"
    )
    parser.add_argument(
        "--dump",
        default=str(Path(__file__).parent / "data/curl_dump.txt"),
        help="Path to the curl dump file",
    )
    parser.add_argument(
        "--fail-only", action="store_true", help="Only display failing tests"
    )
    parser.add_argument("--json", action="store_true", help="Write JSON results file")
    parser.add_argument(
        "--timeout", type=float, default=10.0, help="Per-request timeout (s)"
    )
    parser.add_argument(
        "--parallel",
        type=int,
        default=1,
        help="Number of parallel workers (default 1 = sequential)",
    )
    args = parser.parse_args()

    dump_path = Path(args.dump)
    if not dump_path.exists():
        console.print(f"[red]Error:[/red] Curl dump not found: {dump_path}")
        sys.exit(1)

    # ── Header ─────────────────────────────────────────────────────────────
    console.print()
    console.print(
        Panel(
            "[bold cyan]🛸 Hoverfly Smoke-Test Suite[/bold cyan]\n"
            f"[dim]Target:[/dim] [white]{args.url}[/white]   "
            f"[dim]Dump:[/dim] [white]{dump_path.name}[/white]   "
            f"[dim]Workers:[/dim] [white]{args.parallel}[/white]",
            border_style="cyan",
            expand=False,
        )
    )

    # ── Load test cases ────────────────────────────────────────────────────
    cases = load_test_cases(dump_path)
    console.print(
        f"[dim]Loaded [bold]{len(cases)}[/bold] test cases from {dump_path.name}[/dim]"
    )
    console.print()

    # ── Run tests with progress bar ────────────────────────────────────────
    results: list[TestResult] = [None] * len(cases)  # type: ignore[list-item]

    progress = Progress(
        SpinnerColumn(),
        TextColumn("[progress.description]{task.description}"),
        BarColumn(bar_width=40),
        MofNCompleteColumn(),
        TaskProgressColumn(),
        TimeElapsedColumn(),
        console=console,
    )

    with progress:
        task = progress.add_task("[cyan]Running API tests…", total=len(cases))

        if args.parallel == 1:
            with httpx2.Client() as client:
                for case in cases:
                    result = run_case(case, args.url, args.timeout, client)
                    results[case.index - 1] = result
                    icon = (
                        "✓"
                        if result.status == "pass"
                        else ("✗" if result.status == "fail" else "⚠")
                    )
                    color = (
                        "green"
                        if result.status == "pass"
                        else ("red" if result.status == "fail" else "yellow")
                    )
                    progress.update(
                        task,
                        advance=1,
                        description=f"[{color}]{icon}[/{color}] [dim]{case.method[:55]}[/dim]",
                    )
        else:
            with httpx2.Client() as client:
                with ThreadPoolExecutor(max_workers=args.parallel) as executor:
                    future_to_case = {
                        executor.submit(
                            run_case, case, args.url, args.timeout, client
                        ): case
                        for case in cases
                    }
                    for future in as_completed(future_to_case):
                        result = future.result()
                        results[result.case.index - 1] = result
                        icon = (
                            "✓"
                            if result.status == "pass"
                            else ("✗" if result.status == "fail" else "⚠")
                        )
                        color = (
                            "green"
                            if result.status == "pass"
                            else ("red" if result.status == "fail" else "yellow")
                        )
                        progress.update(
                            task,
                            advance=1,
                            description=f"[{color}]{icon}[/{color}] [dim]{result.case.method[:55]}[/dim]",
                        )

    console.print()

    # ── Scoreboard ─────────────────────────────────────────────────────────
    console.print(build_scoreboard(results))
    console.print()

    # ── Failures panel ─────────────────────────────────────────────────────
    failures_panel = build_failures_panel(results)
    if failures_panel:
        console.print(failures_panel)
        console.print()

    # ── Full results table ─────────────────────────────────────────────────
    table = build_summary_table(results, fail_only=args.fail_only)
    console.print(table)

    # ── JSON export ────────────────────────────────────────────────────────
    if args.json:
        out_path = Path(__file__).parent / "hoverfly_test_results.json"
        export = []
        for r in results:
            export.append(
                {
                    "index": r.case.index,
                    "method": r.case.method,
                    "status": r.status,
                    "http_code": r.http_code,
                    "elapsed_ms": round(r.elapsed_ms, 2),
                    "error": r.error_msg or None,
                    "response": r.response,
                }
            )
        out_path.write_text(json.dumps(export, indent=2))
        console.print(f"\n[dim]Results written to:[/dim] [white]{out_path}[/white]")

    # ── Exit code ──────────────────────────────────────────────────────────
    failed_count = sum(
        1 for r in results if r.status in ("fail", "error") and r.detail != "SKIP"
    )
    if failed_count:
        console.print(f"\n[bold red]{failed_count} test(s) failed.[/bold red]")
        sys.exit(1)
    else:
        console.print("\n[bold green]All tests passed! 🎉[/bold green]")
        sys.exit(0)


if __name__ == "__main__":
    main()
