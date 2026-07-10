#!/usr/bin/env python3
"""
Agent Template 2: Agent Loop Failure
Task: Find the fix for a ROCm OOM error (gets stuck in loop).
"""
import time
from agent_base import AgentTracer


def main():
    tracer = AgentTracer()

    # Start agent run
    tracer.start_agent_run(
        agent_name="Research Agent (Loop Failure)",
        task="Find the fix for a ROCm OOM error"
    )

    time.sleep(0.3)

    # Step 1: TOOL_CALL → Search docs (first attempt)
    tracer.tool_call(
        tool_name="search_docs",
        args='{"query": "ROCm OOM error"}',
        result="Found 12 documents about ROCm OOM errors, but results are too general",
        latency_ms=298
    )

    time.sleep(0.3)

    # Step 2: TOOL_CALL → Search docs (same query, second attempt)
    tracer.tool_call(
        tool_name="search_docs",
        args='{"query": "ROCm OOM error"}',
        result="Found 12 documents about ROCm OOM errors, but results are too general",
        latency_ms=287
    )

    time.sleep(0.3)

    # Step 3: TOOL_CALL → Search docs (same query, third attempt)
    tracer.tool_call(
        tool_name="search_docs",
        args='{"query": "ROCm OOM error"}',
        result="Found 12 documents about ROCm OOM errors, but results are too general",
        latency_ms=294
    )

    time.sleep(0.3)

    # Step 4: TOOL_CALL → Search docs (same query, fourth attempt)
    tracer.tool_call(
        tool_name="search_docs",
        args='{"query": "ROCm OOM error"}',
        result="Found 12 documents about ROCm OOM errors, but results are too general",
        latency_ms=301
    )

    time.sleep(0.3)

    # Step 5: TOOL_CALL → Search docs (same query, fifth attempt - max retries)
    tracer.tool_call(
        tool_name="search_docs",
        args='{"query": "ROCm OOM error"}',
        result="Found 12 documents about ROCm OOM errors, but results are too general",
        latency_ms=289
    )

    time.sleep(0.2)

    # Step 6: ERROR → Agent detected in infinite loop
    tracer.error(
        error_type="AGENT_LOOP",
        message="Agent detected making repeated identical tool calls without progress. Called search_docs with same query 5 times. Loop breaker triggered."
    )

    # Finish with failure
    tracer.finish(status="failed", failure_type="AGENT_LOOP")

    print("\n🔴 AGENT FAILURE: Infinite loop detected!")
    print("The agent got stuck calling the same tool repeatedly without making progress.")
    print("This is a common failure mode in autonomous agents.")


if __name__ == "__main__":
    main()
