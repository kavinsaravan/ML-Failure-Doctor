"""
Example: Track a real LangChain agent with CrashLens
"""

import sys
sys.path.insert(0, '..')

from crashlens.agent_tracker import AgentTracer

# Initialize agent tracker
tracer = AgentTracer("https://invigorating-empathy-production-dee5.up.railway.app")

# Start tracking an agent run
tracer.start_agent_run(
    agent_name="ResearchAgent",
    task="Research the best practices for fine-tuning LLMs on AMD GPUs"
)

try:
    # Step 1: Planning
    tracer.model_call(
        prompt="Plan research on AMD GPU fine-tuning",
        response="I will search documentation, compare methods, and summarize findings",
        tokens=50,
        latency_ms=234
    )
    
    # Step 2: Tool call - search
    tracer.tool_call(
        tool_name="web_search",
        args_json='{"query": "AMD GPU LLM fine-tuning best practices"}',
        result_json='{"results": ["ROCm optimization tips", "Mixed precision training"]}',
        latency_ms=1200
    )
    
    # Step 3: Tool call - read documentation
    tracer.tool_call(
        tool_name="read_docs",
        args_json='{"url": "https://rocm.docs.amd.com"}',
        result_json='{"content": "ROCm supports PyTorch..."}',
        latency_ms=800
    )
    
    # Step 4: Synthesize findings
    tracer.model_call(
        prompt="Summarize AMD GPU fine-tuning best practices",
        response="1. Use ROCm 5.7+\n2. Enable mixed precision\n3. Use gradient checkpointing",
        tokens=120,
        latency_ms=456
    )
    
    # Final response
    tracer.final_response(
        "Based on research: Best practices include using ROCm 5.7+, "
        "enabling mixed precision training, and gradient checkpointing for memory efficiency."
    )
    
    # Mark as completed
    tracer.finish(status="completed")
    
    print("Agent run tracked successfully!")
    print(f"Check dashboard for agent run details")
    
except Exception as e:
    # Track error
    tracer.error("EXECUTION_ERROR", str(e))
    tracer.finish(status="failed", failure_type="EXECUTION_ERROR")
    raise
