#!/usr/bin/env python3
"""
Tool Loop Agent
Simulates an AI agent that gets stuck in an infinite tool-calling loop.
"""

import time
import sys
import json

def main():
    print("Starting AI Agent execution...")
    print("Agent: ReAct Agent with Tool Calling")
    time.sleep(0.5)

    trace_events = []

    # Step 1
    print("\n=== Step 1 ===")
    print("Thought: I need to check the weather")
    print("Action: call_tool('get_weather', {'location': 'San Francisco'})")
    trace_events.append({
        "step": 1,
        "thought": "I need to check the weather",
        "action": "get_weather",
        "args": {"location": "San Francisco"}
    })
    time.sleep(0.3)
    print("Result: Error - API key not configured")

    # Step 2 - starts loop
    print("\n=== Step 2 ===")
    print("Thought: I should try getting the weather again")
    print("Action: call_tool('get_weather', {'location': 'San Francisco'})")
    trace_events.append({
        "step": 2,
        "thought": "I should try getting the weather again",
        "action": "get_weather",
        "args": {"location": "San Francisco"}
    })
    time.sleep(0.3)
    print("Result: Error - API key not configured")

    # Infinite loop detection
    for i in range(3, 12):
        print(f"\n=== Step {i} ===")
        print("Thought: Maybe it will work this time")
        print("Action: call_tool('get_weather', {'location': 'San Francisco'})")
        trace_events.append({
            "step": i,
            "thought": "Maybe it will work this time",
            "action": "get_weather",
            "args": {"location": "San Francisco"}
        })
        time.sleep(0.2)
        print("Result: Error - API key not configured")

    # Error after max iterations
    print("\n=== AGENT FAILED ===")
    print("ERROR: Agent stuck in tool-calling loop")
    print("Failed after 11 steps with no progress")
    print("Repeated tool: get_weather (called 11 times)")
    print("Same error: API key not configured")
    print("\nAgent trace:")
    print(json.dumps(trace_events[-3:], indent=2))

    return 1

if __name__ == "__main__":
    sys.exit(main())
