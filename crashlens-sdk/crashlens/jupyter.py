"""
CrashLens Jupyter Notebook Integration

Provides IPython magic commands, rich display, and inline visualizations for Jupyter notebooks.
"""

import os
import json
import logging
from html import escape
from .workload_tracker import WorkloadTracker
import time
import requests
from typing import Optional, Dict, Any
from contextlib import contextmanager

# Check if running in Jupyter
try:
    from IPython import get_ipython
    from IPython.core.magic import Magics, line_magic, cell_magic, magics_class
    from IPython.display import display, HTML
    import pandas as pd
    JUPYTER_AVAILABLE = True
except ImportError:
    JUPYTER_AVAILABLE = False
    get_ipython = None
    Magics = object
    magics_class = lambda x: x
    line_magic = lambda x: x
    cell_magic = lambda x: x


class JupyterWorkloadTracker(WorkloadTracker):
    """
    Enhanced WorkloadTracker for Jupyter notebooks with rich display and inline metrics.

    Features:
    - Real-time progress display
    - Inline GPU metrics visualization
    - Rich HTML output for diagnostics
    - Live log streaming

    Usage:
        from crashlens.jupyter import JupyterWorkloadTracker

        tracker = JupyterWorkloadTracker("https://your-backend.railway.app")

        with tracker.track("Training Model", display_metrics=True):
            model.fit(X_train, y_train, epochs=10)
    """

    def __init__(self, api_url: str, api_key: Optional[str] = None, fireworks_api_key: Optional[str] = None, fireworks_model: Optional[str] = None, device=None):
        super().__init__(api_url, api_key, fireworks_api_key=fireworks_api_key, fireworks_model=fireworks_model, device=device)
        self.in_jupyter = self._check_jupyter()

    def _check_jupyter(self) -> bool:
        """Check if running in Jupyter environment"""
        if not JUPYTER_AVAILABLE:
            return False
        try:
            shell = get_ipython().__class__.__name__
            return shell in ['ZMQInteractiveShell', 'TerminalInteractiveShell']
        except:
            return False

    def _get_workload(self, workload_id: int) -> Dict[str, Any]:
        """Get workload details"""
        response = requests.get(f"{self.api_url}/workloads/{workload_id}", headers=self.headers, timeout=15)
        response.raise_for_status()
        return response.json()

    def _display_metrics(self, workload_id: int):
        """Display GPU metrics inline (Jupyter only)"""
        if not self.in_jupyter:
            return

        try:
            response = requests.get(f"{self.api_url}/workloads/{workload_id}/metrics", headers=self.headers, timeout=15)
            if response.status_code == 200:
                metrics = response.json().get("metrics", "")
                if isinstance(metrics, str):
                    metrics = json.loads(metrics) if metrics else []
                if metrics and isinstance(metrics, list):
                    # Convert to DataFrame for nice display
                    df = pd.DataFrame(metrics)
                    if not df.empty:
                        # Display last 5 metrics
                        display(HTML("<h4>📊 Latest GPU Metrics</h4>"))
                        display(df.tail(5))
        except:
            pass

    def _display_diagnosis(self, diagnosis: Dict[str, Any]):
        """Display diagnosis with rich HTML formatting"""
        # The backend report uses a single formatted recommended_fix string.
        fixes = diagnosis.get("recommended_fix", "").splitlines()
        if not fixes:
            fixes = diagnosis.get("recommended_fixes", [])
        prevention = diagnosis.get("prevention", "")
        if isinstance(prevention, str):
            prevention = [prevention] if prevention else []
        source_label = "AI-assisted" if diagnosis.get("source") == "ai" else "Rule-based"
        if not self.in_jupyter:
            print(f"Root Cause: {escape(str(diagnosis.get('root_cause', 'Unknown')))}")
            print(f"\nRecommended Fixes:")
            for fix in fixes:
                print(f"  • {fix}")
            return

        # Rich HTML display for Jupyter
        html = f"""
        <div style="border: 2px solid #e74c3c; border-radius: 8px; padding: 20px; margin: 10px 0; background-color: #fdf2f2;">
            <h3 style="color: #c0392b; margin-top: 0;">🔍 {source_label} Diagnosis Report</h3>

            <div style="margin: 15px 0;">
                <strong style="color: #2c3e50;">Root Cause:</strong>
                <p style="margin: 5px 0 0 20px; color: #34495e;">{escape(str(diagnosis.get('root_cause', 'Unknown')))}</p>
            </div>

            <div style="margin: 15px 0;">
                <strong style="color: #2c3e50;">Evidence:</strong>
                <ul style="margin: 5px 0 0 20px; color: #34495e;">
                    {''.join([f'<li><code>{escape(str(e))}</code></li>' for e in diagnosis.get('evidence', [])])}
                </ul>
            </div>

            <div style="margin: 15px 0;">
                <strong style="color: #2c3e50;">✅ Recommended Fixes:</strong>
                <ol style="margin: 5px 0 0 20px; color: #34495e;">
                    {''.join([f'<li>{escape(str(f))}</li>' for f in fixes])}
                </ol>
            </div>

            <div style="margin: 15px 0;">
                <strong style="color: #2c3e50;">Prevention:</strong>
                <ul style="margin: 5px 0 0 20px; color: #34495e;">
                    {''.join([f'<li>{escape(str(p))}</li>' for p in prevention])}
                </ul>
            </div>

            <div style="margin: 15px 0; padding: 10px; background-color: #fff; border-radius: 4px;">
                <strong style="color: #2c3e50;">Safe to Retry:</strong>
                <span style="color: {'#27ae60' if diagnosis.get('safe_to_retry') else '#e74c3c'}; font-weight: bold;">
                    {'✓ Yes' if diagnosis.get('safe_to_retry') else '✗ No'}
                </span>
            </div>

            <div style="margin-top: 10px; font-size: 0.9em; color: #7f8c8d;">
                Heuristic log-match score: {diagnosis.get('confidence', 0) * 100:.1f}% (not measured accuracy)
            </div>
        </div>
        """
        display(HTML(html))

    def _display_progress(self, workload_id: int, name: str, elapsed: float):
        """Display live progress indicator"""
        if not self.in_jupyter:
            return

        status_emoji = "🔄"
        html = f"""
        <div style="border: 2px solid #3498db; border-radius: 8px; padding: 15px; margin: 10px 0; background-color: #ebf5fb;">
            <h4 style="color: #2980b9; margin: 0;">{status_emoji} Workload Running</h4>
            <p style="margin: 10px 0 0 0;"><strong>Name:</strong> {escape(str(name))}</p>
            <p style="margin: 5px 0 0 0;"><strong>Workload ID:</strong> {workload_id}</p>
            <p style="margin: 5px 0 0 0;"><strong>Elapsed:</strong> {elapsed:.1f}s</p>
        </div>
        """
        display(HTML(html))

    def _display_success(self, name: str, runtime: float, workload_id: int):
        """Display success message"""
        if not self.in_jupyter:
            print(f"✓ Workload '{escape(str(name))}' completed successfully in {runtime:.1f}s")
            return

        html = f"""
        <div style="border: 2px solid #27ae60; border-radius: 8px; padding: 15px; margin: 10px 0; background-color: #eafaf1;">
            <h4 style="color: #27ae60; margin: 0;">✓ Workload Completed Successfully</h4>
            <p style="margin: 10px 0 0 0;"><strong>Name:</strong> {escape(str(name))}</p>
            <p style="margin: 5px 0 0 0;"><strong>Runtime:</strong> {runtime:.1f}s</p>
            <p style="margin: 5px 0 0 0;"><strong>Workload ID:</strong> {workload_id}</p>
            <p style="margin: 10px 0 0 0; font-size: 0.9em; color: #7f8c8d;">
                Open workload #{workload_id} in your CrashLens dashboard.
            </p>
        </div>
        """
        display(HTML(html))

    def _display_failure(self, name: str, runtime: float, workload_id: int, error: str):
        """Display failure message"""
        if not self.in_jupyter:
            print(f"✗ Workload '{escape(str(name))}' failed after {runtime:.1f}s")
            print(f"Error: {error}")
            return

        html = f"""
        <div style="border: 2px solid #e74c3c; border-radius: 8px; padding: 15px; margin: 10px 0; background-color: #fdf2f2;">
            <h4 style="color: #c0392b; margin: 0;">✗ Workload Failed</h4>
            <p style="margin: 10px 0 0 0;"><strong>Name:</strong> {escape(str(name))}</p>
            <p style="margin: 5px 0 0 0;"><strong>Runtime:</strong> {runtime:.1f}s</p>
            <p style="margin: 5px 0 0 0;"><strong>Workload ID:</strong> {workload_id}</p>
            <p style="margin: 10px 0 0 0; font-size: 0.9em; color: #7f8c8d;">
                Run diagnosis: <code>tracker.diagnose({workload_id})</code>
            </p>
        </div>
        """
        display(HTML(html))

    @contextmanager
    def track(self, name: str, display_metrics: bool = True, auto_diagnose: bool = False):
        """
        Track a workload with rich Jupyter display

        Args:
            name: Workload name
            display_metrics: Show GPU metrics inline (default: True)
            auto_diagnose: Automatically run diagnosis on failure (default: False)

        Example:
            with tracker.track("Training ResNet", display_metrics=True):
                model.fit(X_train, y_train)
        """
        workload_id = None
        start = time.monotonic()
        try:
            with super().track(name) as workload_id:
                if self.in_jupyter:
                    self._display_progress(workload_id, name, 0)
                yield workload_id
        except BaseException as error:
            if workload_id is not None:
                if isinstance(error, SystemExit) and error.code in (None, 0):
                    self._display_success(name, time.monotonic() - start, workload_id)
                    raise
                try:
                    self._display_failure(name, time.monotonic() - start, workload_id, str(error))
                except Exception:
                    logging.getLogger(__name__).warning("CrashLens display failed", exc_info=True)
                if auto_diagnose:
                    try:
                        self._display_diagnosis(self.diagnose(workload_id, display=False))
                    except Exception:
                        logging.getLogger(__name__).warning("CrashLens diagnosis failed", exc_info=True)
            raise
        else:
            self._display_success(name, time.monotonic() - start, workload_id)
            if display_metrics and self.in_jupyter:
                self._display_metrics(workload_id)

    def diagnose(self, workload_id: int, display: bool = True) -> Dict[str, Any]:
        """
        Run AI diagnosis on a failed workload

        Args:
            workload_id: ID of the workload to diagnose
            display: Whether to display rich HTML output (default: True)

        Returns:
            dict with root_cause, recommended_fixes, evidence, etc.
        """
        diagnosis = super().diagnose(workload_id)

        if display:
            self._display_diagnosis(diagnosis)

        return diagnosis

    def show_workload(self, workload_id: int):
        """Display workload details in rich HTML format"""
        workload = self._get_workload(workload_id)

        if not self.in_jupyter:
            print(f"Workload ID: {workload['id']}")
            print(f"Name: {escape(str(workload['name']))}")
            print(f"Status: {workload['status']}")
            print(f"Runtime: {workload.get('runtime_seconds', 0)}s")
            return

        status_color = {
            'succeeded': '#27ae60',
            'failed': '#e74c3c',
            'running': '#3498db',
            'pending': '#f39c12'
        }.get(workload['status'], '#95a5a6')

        html = f"""
        <div style="border: 2px solid {status_color}; border-radius: 8px; padding: 20px; margin: 10px 0;">
            <h3 style="color: {status_color}; margin-top: 0;">Workload #{workload['id']}: {escape(str(workload['name']))}</h3>

            <table style="width: 100%; border-collapse: collapse;">
                <tr>
                    <td style="padding: 8px; font-weight: bold; width: 150px;">Status:</td>
                    <td style="padding: 8px; color: {status_color};">{workload['status'].upper()}</td>
                </tr>
                <tr style="background-color: #f8f9fa;">
                    <td style="padding: 8px; font-weight: bold;">Type:</td>
                    <td style="padding: 8px;">{workload.get('type', 'N/A')}</td>
                </tr>
                <tr>
                    <td style="padding: 8px; font-weight: bold;">Runtime:</td>
                    <td style="padding: 8px;">{workload.get('runtime_seconds', 0)}s</td>
                </tr>
                <tr style="background-color: #f8f9fa;">
                    <td style="padding: 8px; font-weight: bold;">Exit Code:</td>
                    <td style="padding: 8px;">{workload.get('exit_code', 'N/A')}</td>
                </tr>
                <tr>
                    <td style="padding: 8px; font-weight: bold;">Failure Type:</td>
                    <td style="padding: 8px;">{workload.get('failure_type', 'N/A')}</td>
                </tr>
                <tr style="background-color: #f8f9fa;">
                    <td style="padding: 8px; font-weight: bold;">Wasted GPU-sec:</td>
                    <td style="padding: 8px;">{workload.get('wasted_gpu_seconds', 0)}</td>
                </tr>
            </table>
        </div>
        """
        display(HTML(html))


# IPython Magic Commands
if JUPYTER_AVAILABLE:
    @magics_class
    class CrashLensMagics(Magics):
        """
        IPython magic commands for CrashLens

        Usage in Jupyter:
            %load_ext crashlens.jupyter
            %crashlens_init https://your-backend.railway.app

            %%crashlens_track "Training Model"
            model.fit(X_train, y_train)
        """

        tracker: Optional[JupyterWorkloadTracker] = None

        @line_magic
        def crashlens_init(self, line):
            """
            Initialize CrashLens tracker

            Usage:
                %crashlens_init https://your-backend.railway.app

            The API key is read from the CRASHLENS_API_KEY environment variable.
            Set it before starting Jupyter:
                export CRASHLENS_API_KEY=your_key
            """
            api_url = line.strip()
            if not api_url:
                print("❌ Please provide API URL: %crashlens_init https://your-backend.railway.app")
                return

            # Get API key from environment, never from cell source
            api_key = os.getenv("CRASHLENS_API_KEY")

            self.tracker = JupyterWorkloadTracker(api_url, api_key, fireworks_api_key=os.getenv("FIREWORKS_API_KEY"), fireworks_model=os.getenv("FIREWORKS_MODEL"))
            display(HTML(f"""
            <div style="border: 2px solid #27ae60; border-radius: 8px; padding: 15px; margin: 10px 0; background-color: #eafaf1;">
                <h4 style="color: #27ae60; margin: 0;">✓ CrashLens Initialized</h4>
                <p style="margin: 10px 0 0 0;">Connected to: <code>{api_url}</code></p>
                <p style="margin: 10px 0 0 0;">Auth: <code>{'Enabled' if api_key else 'Disabled (set CRASHLENS_API_KEY)'}</code></p>
                <p style="margin: 10px 0 0 0;">Use <code>%%crashlens_track "Job Name"</code> to track cells</p>
            </div>
            """))

        @cell_magic
        def crashlens_track(self, line, cell):
            """
            Track a cell as a workload

            Usage:
                %%crashlens_track "Training Model"
                model.fit(X_train, y_train)
            """
            if self.tracker is None:
                print("❌ Please initialize CrashLens first: %crashlens_init <api_url>")
                return

            name = line.strip().strip('"').strip("'")
            if not name:
                name = "Jupyter Cell"

            # Execute cell within tracking context
            with self.tracker.track(name, display_metrics=True, auto_diagnose=True):
                result = self.shell.run_cell(cell)
                result.raise_error()

        @line_magic
        def crashlens_diagnose(self, line):
            """
            Diagnose a workload

            Usage:
                %crashlens_diagnose 123
            """
            if self.tracker is None:
                print("❌ Please initialize CrashLens first: %crashlens_init <api_url>")
                return

            try:
                workload_id = int(line.strip())
                self.tracker.diagnose(workload_id)
            except ValueError:
                print("❌ Please provide a valid workload ID: %crashlens_diagnose 123")

        @line_magic
        def crashlens_show(self, line):
            """
            Show workload details

            Usage:
                %crashlens_show 123
            """
            if self.tracker is None:
                print("❌ Please initialize CrashLens first: %crashlens_init <api_url>")
                return

            try:
                workload_id = int(line.strip())
                self.tracker.show_workload(workload_id)
            except ValueError:
                print("❌ Please provide a valid workload ID: %crashlens_show 123")


def load_ipython_extension(ipython):
    """
    Load the CrashLens Jupyter extension

    Usage in Jupyter notebook:
        %load_ext crashlens.jupyter
    """
    if JUPYTER_AVAILABLE:
        ipython.register_magics(CrashLensMagics)
        display(HTML("""
        <div style="border: 2px solid #3498db; border-radius: 8px; padding: 20px; margin: 10px 0; background-color: #ebf5fb;">
            <h3 style="color: #2980b9; margin-top: 0;">🔍 CrashLens Jupyter Extension Loaded</h3>
            <p style="margin: 10px 0;">Available magic commands:</p>
            <ul style="margin: 10px 0;">
                <li><code>%crashlens_init &lt;api_url&gt;</code> - Initialize tracker</li>
                <li><code>%%crashlens_track "Job Name"</code> - Track a cell</li>
                <li><code>%crashlens_diagnose &lt;workload_id&gt;</code> - Get AI diagnosis</li>
                <li><code>%crashlens_show &lt;workload_id&gt;</code> - Show workload details</li>
            </ul>
            <p style="margin: 10px 0 0 0; font-size: 0.9em; color: #7f8c8d;">
                Start by running: <code>%crashlens_init https://your-backend.railway.app</code>
            </p>
        </div>
        """))
    else:
        print("⚠️  IPython not available. Install with: pip install ipython jupyter")
