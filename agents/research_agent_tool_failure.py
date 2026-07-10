#!/usr/bin/env python3
"""
Agent Template 3: Tool Failure
Task: Analyze failed ML job logs (tool call fails).
"""
import time
from agent_base import AgentTracer


def main():
    tracer = AgentTracer()

    # Start agent run
    tracer.start_agent_run(
        agent_name="Log Analysis Agent",
        task="Analyze failed ML job logs from workload-456"
    )

    time.sleep(0.4)

    # Step 1: MODEL_CALL → Plan task
    tracer.model_call(
        prompt="Task: Analyze failed ML job logs from workload-456. Create an analysis plan.",
        response="Plan: 1) Read the job logs from the workload, 2) Extract error messages and stack traces, 3) Identify root cause, 4) Provide recommendations",
        tokens=120,
        latency_ms=654
    )

    time.sleep(0.4)

    # Step 2: TOOL_CALL → Read logs (this will fail)
    tracer.log_step(
        step_type="TOOL_CALL",
        name="read_logs",
        input_data='{"workload_id": 456, "file": "missing-file.log"}',
        output_data="ERROR: FileNotFoundError: Log file 'missing-file.log' not found in workload 456",
        status="failed",
        latency_ms=127
    )

    tracer.total_tool_calls += 1

    time.sleep(0.2)

    # Step 3: ERROR → Tool call failure
    tracer.error(
        error_type="TOOL_CALL_FAILURE",
        message="Tool 'read_logs' failed: FileNotFoundError - Log file 'missing-file.log' not found. Agent cannot proceed without access to logs."
    )

    # Finish with failure
    tracer.finish(status="failed", failure_type="TOOL_CALL_FAILURE")

    print("\n🔴 AGENT FAILURE: Tool call failed!")
    print("The agent attempted to read logs that don't exist.")
    print("This shows CrashLens can diagnose broken tool usage in agents.")


if __name__ == "__main__":
    main()
