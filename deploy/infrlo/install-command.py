#!/usr/bin/env python3
"""Infrlo Build Command: download the verified, already-compiled AhB Linux release.

No pip, Node.js, Go, Rust, sudo, Docker, or credentials are needed at build time.
"""
import hashlib
import os
from pathlib import Path, PurePosixPath
import platform
import re
import shutil
import sys
import tarfile
import tempfile
from urllib.request import Request, urlopen

RELEASE = "linux-16bf4fd6593e"
REPO = Path(__file__).resolve().parents[2]
DEST = REPO / ".infrlo-native" / "AhB"


def download(url, destination):
    if not url.startswith("https://github.com/Tsenjii/AhB/releases/download/"):
        raise ValueError("unexpected download URL")
    request = Request(url, headers={"User-Agent": "AhB-Infrlo-release-installer"})
    with urlopen(request, timeout=70) as upstream, open(destination, "wb") as file:
        shutil.copyfileobj(upstream, file, length=1024 * 1024)


def install():
    arch = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(
        platform.machine().lower()
    )
    if not arch or platform.system() != "Linux":
        raise RuntimeError("Infrlo command mode requires Linux amd64 or arm64")
    archive_name = "AhB_linux_%s.tar.gz" % arch
    base = "https://github.com/Tsenjii/AhB/releases/download/" + RELEASE
    if DEST.joinpath("bin", "hubd").is_file():
        print("Verified AhB package already staged at", DEST)
        return
    with tempfile.TemporaryDirectory(prefix="ahb-infrlo-") as temp:
        tempdir = Path(temp)
        archive = tempdir / archive_name
        checksum = tempdir / (archive_name + ".sha256")
        print("Fetching pinned AhB Linux release", RELEASE, arch, flush=True)
        download(base + "/" + archive_name, archive)
        download(base + "/" + archive_name + ".sha256", checksum)
        expected_file = checksum.read_text().strip().split()
        if len(expected_file) < 2 or expected_file[1].lstrip("*") != archive_name:
            raise RuntimeError("release checksum has an unexpected filename")
        expected = expected_file[0]
        if not re.fullmatch(r"[a-f0-9]{64}", expected):
            raise RuntimeError("invalid release SHA-256")
        digest = hashlib.sha256()
        with archive.open("rb") as file:
            for block in iter(lambda: file.read(1024 * 1024), b""):
                digest.update(block)
        if digest.hexdigest() != expected:
            raise RuntimeError("native Linux archive SHA-256 mismatch")
        root = tempdir / "unpacked"
        root.mkdir()
        with tarfile.open(archive, "r:gz") as bundle:
            for member in bundle.getmembers():
                name = PurePosixPath(member.name)
                if (
                    member.name.startswith("/")
                    or ".." in name.parts
                    or not name.parts
                    or name.parts[0] != "AhB"
                    or not (member.isfile() or member.isdir())
                ):
                    raise RuntimeError("unsafe entry in native Linux release")
            bundle.extractall(root)
        staged = root / "AhB"
        if not (staged / "bin" / "hubd").is_file():
            raise RuntimeError("missing native Hub executable")
        DEST.parent.mkdir(parents=True, exist_ok=True)
        if DEST.exists():
            raise RuntimeError("incomplete package already present: remove manually for a fresh test")
        shutil.move(str(staged), str(DEST))
    print("Pinned Linux package installed for Infrlo command runner")


if __name__ == "__main__":
    try:
        install()
    except Exception as exc:
        print("AhB Infrlo build failed:", exc, file=sys.stderr)
        sys.exit(1)
