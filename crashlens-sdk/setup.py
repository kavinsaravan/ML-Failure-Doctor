from setuptools import setup, find_packages

setup(
    name="crashlens",
    version="0.1.0",
    description="CrashLens SDK for tracking ML workloads",
    author="Kavin Saravan",
    packages=find_packages(),
    install_requires=[
        "requests>=2.28.0",
    ],
    extras_require={
        "gpu": [
            "nvidia-ml-py>=11.0.0",  # For NVIDIA GPU metrics
        ],
        "jupyter": [
            "ipython>=7.0.0",
            "jupyter>=1.0.0",
            "pandas>=1.3.0",
            "notebook>=6.0.0",
        ],
        "all": [
            "nvidia-ml-py>=11.0.0",
            "ipython>=7.0.0",
            "jupyter>=1.0.0",
            "pandas>=1.3.0",
            "notebook>=6.0.0",
        ],
    },
    python_requires=">=3.7",
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
    ],
)
