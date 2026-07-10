#!/usr/bin/env python3
"""
Agent Template 1: Successful Research Agent
Task: Research common fixes for HIP out-of-memory errors.
"""
import time
from agent_base import AgentTracer


def main():
    tracer = AgentTracer()

    # Start agent run
    tracer.start_agent_run(
        agent_name="Research Agent",
        task="Research common fixes for HIP out-of-memory errors"
    )

    time.sleep(0.5)

    # Step 1: MODEL_CALL → Plan task
    tracer.model_call(
        prompt="Task: Research common fixes for HIP out-of-memory errors. Create a research plan.",
        response="Plan: 1) Search AMD ROCm documentation for HIP memory errors, 2) Search for batch size optimization techniques, 3) Compile findings into actionable recommendations",
        tokens=150,
        latency_ms=823
    )

    time.sleep(0.5)

    # Step 2: TOOL_CALL → Search docs for HIP OOM
    tracer.tool_call(
        tool_name="search_docs",
        args='{"query": "HIP out of memory ROCm"}',
        result="""Found 3 relevant documents:
        1. ROCm Documentation: HIP memory management best practices
        2. AMD GPU Optimization Guide: Memory allocation strategies
        3. Common HIP errors and solutions""",
        latency_ms=342
    )

    time.sleep(0.5)

    # Step 3: TOOL_CALL → Search for optimization techniques
    tracer.tool_call(
        tool_name="search_docs",
        args='{"query": "reduce batch size mixed precision ROCm"}',
        result="""Found 5 relevant documents:
        1. Reducing batch size for AMD GPUs
        2. Mixed precision training with ROCm
        3. Gradient accumulation techniques
        4. Memory-efficient training strategies
        5. ROCm PyTorch memory optimization""",
        latency_ms=389
    )

    time.sleep(0.5)

    # Step 4: MODEL_CALL → Summarize findings
    tracer.model_call(
        prompt="Summarize the research findings on HIP out-of-memory errors and provide recommendations.",
        response="""Summary: HIP out-of-memory errors on AMD GPUs typically occur when model size + batch size exceeds VRAM capacity.

Key Findings:
- ROCm 5.7+ includes improved memory management
- Mixed precision (FP16) can reduce memory by 50%
- Gradient checkpointing trades compute for memory
- Batch size should be tuned based on GPU memory

Recommendations:
1. Start with smaller batch sizes (16-32)
2. Enable mixed precision training
3. Use gradient checkpointing for large models
4. Monitor memory with rocm-smi during training
5. Consider gradient accumulation to simulate larger batches""",
        tokens=250,
        latency_ms=1245
    )

    time.sleep(0.5)

    # Step 5: FINAL_RESPONSE
    tracer.final_response("""Research Complete: HIP Out-of-Memory Error Fixes

Recommended Fixes:
1. Reduce batch size to 16-32 (immediate fix)
2. Enable mixed precision training (FP16) - reduces memory by ~50%
3. Implement gradient checkpointing for large models
4. Use gradient accumulation to simulate larger batches
5. Monitor GPU memory with `rocm-smi --showmeminfo vram`

Additional Resources:
- AMD ROCm Documentation: docs.amd.com/rocm
- PyTorch ROCm support: pytorch.org/get-started/locally/
- Memory optimization guide: github.com/AMD/ROCm/wiki

Estimated impact: Can reduce memory usage by 40-60% with these techniques.""")

    # Finish successfully
    tracer.finish(status="completed")


if __name__ == "__main__":
    main()
