"""
CrashLens Python SDK
Track ML workloads with automatic failure reporting

For Jupyter notebook integration, use:
    from crashlens.jupyter import JupyterWorkloadTracker

Or load the magic commands extension:
    %load_ext crashlens.jupyter
"""

from .workload_tracker import WorkloadTracker

__version__ = "0.1.0"
__all__ = ["WorkloadTracker"]

# Jupyter integration is optional (imported on-demand)
try:
    from . import jupyter
    __all__.append("jupyter")
except ImportError:
    # IPython/Jupyter not installed, that's okay
    pass
