"""
Base agent helper for reporting execution traces to CrashLens.
"""
import requests
import time
import uuid
from typing import Optional


class AgentTracer:
    """Helper class to report agent execution to CrashLens observability platform."""

    def __init__(self, backend_url: str = "http://localhost:8080"):
        self.backend_url = backend_url
        self.workload_id = None
        self.agent_run_id = None
        self.agent_name = None
        self.task = None
        self.step_index = 0
        self.total_tool_calls = 0
        self.total_model_calls = 0
        self.total_tokens = 0
        self.total_latency_ms = 0
        self.steps = []

    def start_agent_run(self, agent_name: str, task: str):
        """Create workload and agent run."""
        self.agent_name = agent_name
        self.task = task
        self.agent_run_id = str(uuid.uuid4())

        # Create workload
        workload_data = {
            "name": f"Agent Run: {agent_name}",
            "type": "AGENT_RUN",
            "status": "running"
        }

        response = requests.post(f"{self.backend_url}/workloads", json=workload_data)
        if response.status_code == 201:
            self.workload_id = response.json()["id"]
            print(f"Created workload {self.workload_id}")
        else:
            raise Exception(f"Failed to create workload: {response.text}")

        # Create agent run
        agent_run_data = {
            "id": self.agent_run_id,
            "workload_id": str(self.workload_id),
            "agent_name": agent_name,
            "task": task,
            "status": "running",
            "total_tool_calls": 0,
            "total_model_calls": 0,
            "total_tokens": 0,
            "total_latency_ms": 0
        }

        response = requests.post(f"{self.backend_url}/agent-runs", json=agent_run_data)
        if response.status_code != 201:
            print(f"Warning: Failed to create agent run: {response.text}")

    def log_step(self, step_type: str, name: str, input_data: str, output_data: str,
                 status: str = "completed", latency_ms: Optional[int] = None):
        """Log a single agent step."""
        self.step_index += 1

        if step_type == "TOOL_CALL":
            self.total_tool_calls += 1
        elif step_type == "MODEL_CALL":
            self.total_model_calls += 1

        if latency_ms:
            self.total_latency_ms += latency_ms

        step_data = {
            "id": str(uuid.uuid4()),
            "agent_run_id": self.agent_run_id,
            "step_index": self.step_index,
            "step_type": step_type,
            "name": name,
            "input": input_data,
            "output": output_data,
            "status": status,
            "latency_ms": latency_ms
        }

        self.steps.append(step_data)

        # Send to backend
        response = requests.post(f"{self.backend_url}/agent-steps", json=step_data)
        if response.status_code != 201:
            print(f"Warning: Failed to log step: {response.text}")

    def tool_call(self, tool_name: str, args: str, result: str, latency_ms: int = None):
        """Log a tool call step."""
        if latency_ms is None:
            latency_ms = 50 + (self.step_index * 10)  # Simulated latency
        self.log_step("TOOL_CALL", tool_name, args, result, "completed", latency_ms)

    def model_call(self, prompt: str, response: str, tokens: int = 100, latency_ms: int = None):
        """Log a model call step."""
        if latency_ms is None:
            latency_ms = 500 + (tokens * 2)  # Simulated latency based on tokens
        self.total_tokens += tokens
        self.log_step("MODEL_CALL", "gemma_4_26b", prompt, response, "completed", latency_ms)

    def decision(self, question: str, answer: str):
        """Log a decision step."""
        self.log_step("DECISION", "agent_decision", question, answer, "completed", 10)

    def error(self, error_type: str, message: str):
        """Log an error step."""
        self.log_step("ERROR", error_type, "", message, "failed", 5)

    def final_response(self, response: str):
        """Log the final response."""
        self.log_step("FINAL_RESPONSE", "agent_final_response", self.task, response, "completed", 50)

    def finish(self, status: str = "completed", failure_type: Optional[str] = None):
        """Finish the agent run."""
        # Update agent run
        agent_run_update = {
            "status": status,
            "failure_type": failure_type,
            "total_tool_calls": self.total_tool_calls,
            "total_model_calls": self.total_model_calls,
            "total_tokens": self.total_tokens,
            "total_latency_ms": self.total_latency_ms
        }

        response = requests.put(f"{self.backend_url}/agent-runs/{self.agent_run_id}", json=agent_run_update)
        if response.status_code != 200:
            print(f"Warning: Failed to update agent run: {response.text}")

        # Update workload
        workload_update = {
            "status": "failed" if status == "failed" else "succeeded",
            "failure_type": failure_type
        }

        response = requests.put(f"{self.backend_url}/workloads/{self.workload_id}", json=workload_update)
        if response.status_code != 200:
            print(f"Warning: Failed to update workload: {response.text}")

        print(f"Agent run finished: {status}")
        print(f"Workload ID: {self.workload_id}")
        print(f"Agent Run ID: {self.agent_run_id}")
        print(f"Total steps: {self.step_index}")
        print(f"Tool calls: {self.total_tool_calls}, Model calls: {self.total_model_calls}")
