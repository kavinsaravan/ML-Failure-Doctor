# CrashLens Python SDK

Track real ML workloads and AI agents with automatic failure reporting and AI-powered diagnosis.

## Installation

```bash
cd crashlens-sdk
pip install -e .
```

## Quick Start

### 1. Track ML Training Jobs

```python
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://your-backend.railway.app")

# Option A: Context Manager (Recommended)
with tracker.track("Training ResNet-50"):
    model.fit(X_train, y_train, epochs=10)
    # Automatically captures logs, errors, and runtime

# Option B: Decorator
@tracker.track_function("Fine-tuning BERT")
def train():
    trainer.train()
    
train()
```

### 2. Track AI Agents

```python
from crashlens.agent_tracker import AgentTracer

tracer = AgentTracer("https://your-backend.railway.app")

# Start tracking
tracer.start_agent_run("MyAgent", "Research quantum computing")

# Log agent steps
tracer.tool_call("web_search", args_json, result_json, latency_ms=500)
tracer.model_call(prompt, response, tokens=100, latency_ms=800)
tracer.decision("Should I continue?", "Yes")

# Finish
tracer.finish(status="completed")
```

## Real-World Examples

### PyTorch Training

```python
import torch
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://your-backend.railway.app")

with tracker.track("GPT-2 Fine-tuning"):
    model = GPT2LMHeadModel.from_pretrained("gpt2")
    trainer = Trainer(model=model, args=training_args)
    trainer.train()
```

If training fails (OOM, CUDA error, etc.), CrashLens will:
- ✅ Capture the full error traceback
- ✅ Record GPU metrics
- ✅ Classify the failure type
- ✅ Provide AI-powered diagnosis with fixes

### TensorFlow/Keras

```python
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://your-backend.railway.app")

with tracker.track("Image Classification"):
    model.compile(optimizer='adam', loss='sparse_categorical_crossentropy')
    model.fit(train_ds, epochs=10, validation_data=val_ds)
```

### Hugging Face Transformers

```python
from transformers import Trainer
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://your-backend.railway.app")

with tracker.track("BERT Fine-tuning - MRPC"):
    trainer = Trainer(
        model=model,
        args=training_args,
        train_dataset=train_dataset,
        eval_dataset=eval_dataset,
    )
    trainer.train()
```

### LangChain Agents

```python
from langchain.agents import initialize_agent
from crashlens.agent_tracker import AgentTracer

tracer = AgentTracer("https://your-backend.railway.app")

tracer.start_agent_run("CustomerSupport", "Handle user query")

# Your LangChain agent code
agent = initialize_agent(tools, llm, agent="zero-shot-react-description")
response = agent.run("What's the weather?")

# Log each step
tracer.tool_call("get_weather", args, result)
tracer.model_call(prompt, response)
tracer.final_response(response)
tracer.finish("completed")
```

## Features

### Automatic Failure Detection

CrashLens automatically detects:
- **GPU Out of Memory** (CUDA OOM, HIP OOM)
- **Dependency Errors** (Missing packages, version mismatches)
- **Data Path Errors** (Missing datasets, corrupted files)
- **Timeout Issues** (Hung training, deadlocks)
- **ROCm/CUDA Errors** (Driver issues, compatibility problems)

### AI-Powered Diagnosis

After a failure, get instant AI analysis:
```python
# Run diagnosis on a failed workload
diagnosis = tracker.diagnose(workload_id)

print(diagnosis["root_cause"])
# "GPU Out of Memory - batch size too large for available memory"

print(diagnosis["recommended_fixes"])
# 1. Reduce batch size from 32 to 16
# 2. Enable gradient checkpointing
# 3. Use mixed precision (FP16)
# ...
```

### Agent Observability

Track every step of your AI agent:
- Tool calls with inputs/outputs
- Model calls with token counts
- Decision points
- Errors and failures
- Total latency and costs

## Dashboard

View all workloads and agents at:
```
https://your-frontend.vercel.app/dashboard
```

Features:
- Real-time monitoring
- Failure classification
- GPU metrics visualization
- AI-powered recommendations
- Cost tracking (wasted GPU-seconds)

## Integration with Popular Frameworks

### PyTorch Lightning

```python
from pytorch_lightning import Trainer
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://your-backend.railway.app")

with tracker.track("Lightning Training"):
    trainer = Trainer(max_epochs=10)
    trainer.fit(model, datamodule)
```

### Ray Train

```python
from ray.train import ScalingConfig
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://your-backend.railway.app")

with tracker.track("Distributed Training"):
    trainer = TorchTrainer(
        train_func,
        scaling_config=ScalingConfig(num_workers=4)
    )
    result = trainer.fit()
```

### AutoGPTQ (LLM Quantization)

```python
from auto_gptq import AutoGPTQForCausalLM
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://your-backend.railway.app")

with tracker.track("Model Quantization"):
    model = AutoGPTQForCausalLM.from_pretrained(model_name)
    model.quantize(quantize_config)
```

## Environment Variables

```bash
# Optional: Set default API URL
export CRASHLENS_API_URL="https://your-backend.railway.app"
```

Then use simplified initialization:
```python
import crashlens

crashlens.init()  # Uses CRASHLENS_API_URL

with crashlens.track("Training"):
    # Your code
    pass
```

## Advanced Usage

### Manual Workload Creation

```python
tracker = WorkloadTracker(api_url)
workload_id = tracker._create_workload("Custom Job")

try:
    # Your code
    train_model()
    tracker._update_workload(workload_id, "succeeded", runtime_seconds=120)
except Exception as e:
    tracker._update_workload(workload_id, "failed", logs=str(e))
```

### Custom Failure Types

```python
tracker._update_workload(
    workload_id,
    status="failed",
    failure_type="CUSTOM_MEMORY_LEAK",
    logs=error_message
)
```

## Best Practices

1. **Use context managers** - Automatic cleanup and error handling
2. **Add descriptive names** - Makes dashboard navigation easier
3. **Run diagnosis** - After failures, use AI to get actionable fixes
4. **Track costs** - Monitor wasted GPU-seconds to optimize
5. **Integrate early** - Add tracking before production to catch issues

## Examples Directory

See `examples/` for more:
- `pytorch_training.py` - PyTorch model training
- `langchain_agent.py` - LangChain agent tracking
- `distributed_training.py` - Multi-GPU training
- `failure_scenarios.py` - Common failure patterns

## Support

- Dashboard: https://frontend-zeta-eight-92.vercel.app
- Issues: https://github.com/kavinsaravan/ML-Failure-Doctor/issues
