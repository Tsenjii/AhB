#!/usr/bin/env python3
"""Python buildpack discovery entry point for Infrlo custom-command hosting.

The actual implementation stays in deploy/infrlo/run-command.py.
No dependency on Flask, requests, pip libraries, or the Python image builder.
"""
from pathlib import Path
import runpy

if __name__ == "__main__":
    runpy.run_path(
        str(Path(__file__).resolve().parent / "deploy" / "infrlo" / "run-command.py"),
        run_name="__main__",
    )
