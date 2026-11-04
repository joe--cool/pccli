#!/usr/bin/env python3
"""Build released pccli documentation without modifying GitHub or Git refs.

Version snapshots are immutable until their major's released head changes. Builds
happen in private temporary sources extracted from exact tags; promotion starts
only after all version and global pages have built successfully.
"""

import argparse
from dataclasses import dataclass
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import subprocess
import tarfile
import tempfile
from urllib.parse import urlsplit


class PublishError(Exception):
    """A release or saved publication cannot be published safely."""


VERSION = re.compile(r"^(?:pccli-)?v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$")
SAFE_TAG = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._+-]*$")
MAJOR = re.compile(r"^v(?:0|[1-9]\d*)$")
MEDIA_SUFFIXES = {".mp4", ".webm", ".gif", ".png", ".jpg", ".jpeg"}
PLATFORMS = (
    ("linux", "amd64", "Linux x86-64"),
    ("linux", "arm64", "Linux ARM64"),
    ("darwin", "amd64", "macOS Intel"),
    ("darwin", "arm64", "macOS Apple silicon"),
    ("windows", "amd64", "Windows x86-64"),
    ("windows", "arm64", "Windows ARM64"),
)


@dataclass(frozen=True)
class Settings:
    repository_root: Path
    state_dir: Path
    work_dir: Path
    base_path: str
    releases_json: Path
    latest_tag: str
    vhs_image: str
    jekyll_image: str


def run(argv, *, cwd=None, env=None):
    try:
        return subprocess.run(argv, cwd=cwd, env=env, check=True, text=True,
                              stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout.strip()
    except subprocess.CalledProcessError as error:
        # Commands receive only isolated release sources and explicit public inputs.
        # Include build diagnostics so CI failures are actionable.
        detail = (error.stderr or error.stdout or "").strip()
        raise PublishError(f"{argv[0]} failed ({error.returncode}): {detail}") from error


def tag_version(tag):
    if not isinstance(tag, str) or not SAFE_TAG.fullmatch(tag) or ".." in tag:
        raise PublishError(f"unsafe release tag: {tag!r}")
    match = VERSION.fullmatch(tag)
    if not match:
        return None
    return tuple(int(match[i]) for i in (1, 2, 3)), match[4], match[5]


def normalize_base(base):
    if base == "":
        return ""
    if not isinstance(base, str) or not base.startswith("/") or base.endswith("/"):
        raise PublishError("base path must be empty or an absolute URL path without trailing slash")
    if any(not re.fullmatch(r"[A-Za-z0-9_-]+", piece) for piece in base[1:].split("/")):
        raise PublishError("base path contains an unsafe URL segment")
    return base


def public_url(value, description):
    if not isinstance(value, str):
        raise PublishError(f"missing {description} URL")
    parsed = urlsplit(value)
    if parsed.scheme != "https" or not parsed.netloc or parsed.username or parsed.password or parsed.fragment:
        raise PublishError(f"invalid {description} URL")
    return value


def commit_for(repo, tag):
    # Validate before supplying a tag as a Git revision. The refs/tags prefix also
    # prevents a branch with the same spelling from changing the archived source.
    tag_version(tag)
    try:
        commit = run(["git", "-C", str(repo), "rev-parse", "--verify", f"refs/tags/{tag}^{{commit}}"])
    except PublishError as error:
        raise PublishError(f"release tag {tag!r} is not present in the repository") from error
    if not re.fullmatch(r"[0-9a-f]{40,64}", commit):
        raise PublishError(f"invalid commit identity for tag {tag}")
    return commit


def release_plan(releases, latest_tag, base):
    if not isinstance(releases, list):
        raise PublishError("releases JSON must be a GitHub release array")
    tag_version(latest_tag)
    published, by_tag, heads = [], {}, {}
    for release in releases:
        if not isinstance(release, dict):
            raise PublishError("invalid GitHub release record")
        if release.get("draft", False):
            continue
        tag = release.get("tag_name")
        parsed = tag_version(tag)
        if tag in by_tag:
            raise PublishError(f"duplicate release tag: {tag}")
        by_tag[tag] = release
        published.append(release)
        if parsed is None or parsed[1] is not None or parsed[2] is not None or release.get("prerelease", False):
            continue
        version = parsed[0]
        major = f"v{version[0]}"
        # Prefer canonical spelling when a migrated release has both tag forms.
        rank = (version, not tag.startswith("pccli-"))
        if major not in heads or rank > heads[major][0]:
            heads[major] = (rank, tag)
    if not heads:
        raise PublishError("no published stable semantic-version releases found")
    latest_parsed = tag_version(latest_tag)
    if latest_tag not in by_tag or latest_parsed is None:
        raise PublishError("GitHub latest tag is not a published stable release")
    latest_major = f"v{latest_parsed[0][0]}"
    if latest_major not in heads or heads[latest_major][1] != latest_tag:
        raise PublishError("GitHub latest tag must match the selected stable head of its major")
    majors = [{"major": major, "tag": value[1], "docs_url": f"{base}/versions/{major}/"}
              for major, value in sorted(heads.items(), key=lambda item: int(item[0][1:]), reverse=True)]
    entries = []
    for release in sorted(published, key=lambda item: item.get("published_at") or "", reverse=True):
        tag = release["tag_name"]
        parsed = tag_version(tag)
        version = tag.removeprefix("pccli-").removeprefix("v") if parsed else tag
        major = f"v{parsed[0][0]}" if parsed else None
        assets = release.get("assets", [])
        if not isinstance(assets, list):
            raise PublishError(f"invalid asset array for {tag}")
        names = {}
        for asset in assets:
            if not isinstance(asset, dict) or not isinstance(asset.get("name"), str):
                raise PublishError(f"invalid asset for {tag}")
            if asset["name"] in names:
                raise PublishError(f"duplicate asset name in {tag}: {asset['name']}")
            names[asset["name"]] = asset
        downloads = []
        for operating_system, architecture, label in PLATFORMS:
            extension = "zip" if operating_system == "windows" else "tar.gz"
            name = f"pccli_{version}_{operating_system}_{architecture}.{extension}"
            if parsed is None or name not in names:
                continue
            asset = names[name]
            size = asset.get("size", 0)
            if not isinstance(size, int) or isinstance(size, bool) or size < 0:
                raise PublishError(f"invalid asset size for {tag}/{name}")
            downloads.append({"label": label, "os": operating_system, "arch": architecture,
                              "name": name, "url": public_url(asset.get("browser_download_url"), "asset"), "size": size})
        checksum = names.get("checksums.txt")
        entries.append({"tag": tag, "name": release.get("name") or tag, "version": version,
                        "major": major, "published_at": release.get("published_at") or "",
                        "url": public_url(release.get("html_url"), "release"),
                        "notes": release.get("body") or "",
                        "docs_url": f"{base}/versions/{major}/" if major in heads else None,
                        "is_latest": tag == latest_tag, "prerelease": bool(release.get("prerelease", False) or parsed and parsed[1]),
                        "downloads": downloads,
                        "checksum_url": public_url(checksum.get("browser_download_url"), "checksum") if checksum else None})
    return {"latest_tag": latest_tag, "latest_major": latest_major, "majors": majors, "entries": entries}


def read_state(state):
    if not state.exists():
        return {"majors": [], "tags": {}}
    if not state.is_dir() or state.is_symlink():
        raise PublishError("state directory must be a real directory")
    public = [path for path in state.iterdir() if path.name != ".git"]
    metadata = state / "versions.json"
    if not public:
        return {"majors": [], "tags": {}}
    if not metadata.is_file() or metadata.is_symlink():
        raise PublishError("nonempty publication state is missing versions.json; refusing to replace unknown files")
    for path in public:
        if path.is_symlink() or path.is_dir() and any(child.is_symlink() for child in path.rglob("*")):
            raise PublishError("publication state cannot contain symbolic links")
    try:
        saved = json.loads(metadata.read_text())
        majors = saved["majors"]
        tags = saved.get("tags", {})
        if not isinstance(majors, list) or not isinstance(tags, dict):
            raise ValueError("invalid versions metadata")
        seen = set()
        for head in majors:
            if not isinstance(head, dict) or not MAJOR.fullmatch(head.get("major", "")) or head["major"] in seen:
                raise ValueError("invalid major metadata")
            seen.add(head["major"])
            tag_version(head["tag"])
            if not isinstance(head.get("docs_url"), str) or not head["docs_url"]:
                raise ValueError("missing major documentation URL")
            if not re.fullmatch(r"[0-9a-f]{40,64}", head["commit"]):
                raise ValueError("invalid commit metadata")
            if not (state / "versions" / head["major"]).is_dir():
                raise ValueError("missing saved major snapshot")
            tags.setdefault(head["tag"], head["commit"])
        for tag, commit in tags.items():
            tag_version(tag)
            if not re.fullmatch(r"[0-9a-f]{40,64}", commit):
                raise ValueError("invalid saved tag identity")
        return {**saved, "majors": majors, "tags": tags}
    except (ValueError, KeyError, TypeError) as error:
        raise PublishError(f"invalid versions.json: {error}") from error


def archive_tag(repo, tag, destination, workspace, commit=None):
    commit = commit or commit_for(repo, tag)
    names = run(["git", "-C", str(repo), "ls-tree", "-r", "--name-only", "-z", commit]).split("\0")
    if any(PurePosixPath(name).name == ".env" for name in names if name):
        raise PublishError(f"tag {tag} contains a tracked .env; refusing to archive credentials")
    archive = workspace / f"source-{tag}.tar"
    run(["git", "-C", str(repo), "archive", "--format=tar", f"--output={archive}", commit])
    destination.mkdir()
    with tarfile.open(archive) as source:
        for member in source:
            path = PurePosixPath(member.name)
            if path.is_absolute() or ".." in path.parts or not path.parts or path.parts[0] == ".git" or path.name == ".env":
                raise PublishError(f"unsafe or credential-bearing path in tag {tag}: {member.name}")
            # This contributor-only alias is tracked in released tags. It is
            # irrelevant to builds; never follow it or extract any symlink.
            if member.issym() and member.name == "CLAUDE.md" and member.linkname == "AGENTS.md":
                continue
            target = destination.joinpath(*path.parts)
            if member.isdir():
                target.mkdir(parents=True, exist_ok=True)
            elif member.isfile():
                target.parent.mkdir(parents=True, exist_ok=True)
                with source.extractfile(member) as incoming, target.open("wb") as outgoing:
                    shutil.copyfileobj(incoming, outgoing)
                target.chmod(0o755 if member.mode & 0o111 else 0o644)
            else:
                raise PublishError(f"unsupported link or special file in tag {tag}: {member.name}")
    archive.unlink()
    if not (destination / "docs" / "_config.yml").is_file():
        raise PublishError(f"tag {tag} has no documentation configuration")


def ensure_release_navigation(docs):
    """Keep older released layouts connected to current release documentation."""
    layout = docs / "_layouts" / "default.html"
    if layout.is_file():
        text = layout.read_text()
        if all(marker in text for marker in ("site.docs_root", "/releases.html", "/downloads.html", "site.release_version")):
            return
        banner = """
<div class="release-banner" role="note">
  <p>Archived {{ site.release_major | escape }} documentation ({{ site.release_version | escape }}).</p>
  <nav aria-label="Release documentation">
    <a href="{{ site.docs_root | append: '/' | escape }}">Latest documentation</a>
    <a href="{{ site.docs_root | append: '/releases.html' | escape }}">Releases</a>
    <a href="{{ site.docs_root | append: '/downloads.html' | escape }}">Downloads</a>
  </nav>
</div>
"""
        text, count = re.subn(r"(<body\b[^>]*>)", lambda match: match[1] + banner, text, count=1, flags=re.IGNORECASE)
        if count != 1:
            raise PublishError("legacy documentation layout has no body for its release banner")
        layout.write_text(text)


def prepare_legacy_docs(source, head):
    """Repair archival-only references without executing obsolete demo writes."""
    docs = source / "docs"
    if head["tag"].startswith("pccli-v"):
        # Prefix-era tags used the unsuffixed module, including the first v2
        # release. Pin those historical commands without altering /v2 installs.
        install = re.compile(r"(\bgo\s+install\s+)([A-Za-z0-9_./~+-]+)@latest\b")
        for path in docs.rglob("*"):
            if path.is_file() and path.suffix in {".md", ".html"}:
                text = path.read_text()
                text = install.sub(lambda match: match[0] if re.search(r"/v[2-9]\d*/", match[2])
                                   else match[1] + match[2] + "@" + head["tag"], text)
                path.write_text(text)
    ensure_release_navigation(docs)
    if (source / "scripts" / "record-demos.sh").is_file():
        return False
    if head["major"] != "v1":
        raise PublishError(f"tag {head['tag']} has no supported read-only demo recorder")
    docs = source / "docs"
    missing_demo = not (docs / "assets" / "library-demo.gif").is_file()
    image = re.compile(r"<img\b(?=[^>]*\bclass\s*=\s*[\"'][^\"']*\bterminal-demo\b)[^>]*>", re.IGNORECASE)
    for path in docs.rglob("*"):
        if not path.is_file() or path.suffix not in {".md", ".html"}:
            continue
        text = path.read_text()
        text = re.sub(r"(\bgo\s+install\s+[A-Za-z0-9_./~+-]+)@latest\b",
                      lambda match: match[1] + "@" + head["tag"], text)
        if missing_demo:
            text = image.sub("<p>The terminal recording was not archived; use the written examples below.</p>", text)
        path.write_text(text)
    return True


def ensure_search_metadata(source):
    """Use the root as the canonical URL, including on the /latest/ alias."""
    layout = source / "docs" / "_layouts" / "default.html"
    if not layout.is_file():
        return
    text = layout.read_text()
    marker = "<!-- pccli-canonical -->"
    if marker in text:
        return
    block = """<!-- pccli-canonical -->
<link rel="canonical" href="{{ site.url | append: site.docs_root | append: page.url | replace: '/index.html', '/' | escape }}">
<!-- /pccli-canonical -->
"""
    text, count = re.subn(r"</head\s*>", lambda match: block + match[0], text, count=1, flags=re.IGNORECASE)
    if count != 1:
        raise PublishError("documentation layout has no head for search metadata")
    layout.write_text(text)


def exclude_archive_from_search(directory):
    """Stamp built HTML without rebuilding frozen versions or changing media."""
    for path in directory.rglob("*.html"):
        text = path.read_text()
        head = re.search(r"<head\b[^>]*>(.*?)</head\s*>", text, re.IGNORECASE | re.DOTALL)
        if not head:
            raise PublishError(f"archived HTML has no head for indexing policy: {path.name}")
        content = head[1]
        # An archive is intentionally excluded, rather than declared equivalent
        # to potentially different current content. Keep canonical hints separate.
        content = re.sub(r"<link\b(?=[^>]*\brel\s*=\s*[\"']canonical[\"'])[^>]*>\s*", "", content, flags=re.IGNORECASE)
        content = re.sub(r"<meta\b(?=[^>]*\bname\s*=\s*[\"']robots[\"'])[^>]*>\s*", "", content, flags=re.IGNORECASE)
        content = content.rstrip() + '\n<meta name="robots" content="noindex, follow">\n'
        updated = text[:head.start(1)] + content + text[head.end(1):]
        if updated != text:
            path.write_text(updated)


class Builder:
    """External build seam: tests substitute this without rendering or networking."""

    def __init__(self, settings):
        self.settings = settings

    def build_major(self, source, destination, head, dataset):
        if not (source / "scripts" / "record-demos.sh").is_file():
            if head["major"] != "v1":
                raise PublishError(f"tag {head['tag']} has no supported read-only demo recorder")
            self.build_jekyll(source, destination, head, dataset,
                              f"{self.settings.base_path}/versions/{head['major']}")
            return
        binary = source / "bin" / "pccli"
        binary.parent.mkdir(exist_ok=True)
        environment = {**os.environ, "CGO_ENABLED": "0", "GOWORK": "off"}
        module = run(["go", "list", "-m"], cwd=source, env=environment)
        if not re.fullmatch(r"[A-Za-z0-9._~/-]+", module):
            raise PublishError("go list returned an invalid module path")
        version = head["tag"].removeprefix("pccli-").removeprefix("v")
        run(["go", "build", "-buildvcs=false", "-trimpath", "-ldflags", f"-s -w -X {module}/internal/cli.Version={version}",
             "-o", str(binary), "./cmd/pccli"], cwd=source, env=environment)
        command = ["docker", "run", "--rm", "--network", "none", "--user", f"{os.getuid()}:{os.getgid()}",
                   "--entrypoint", "sh", "-v", f"{source}:/vhs", "-w", "/vhs",
                   "-e", "HOME=/tmp/pccli-docs-home", "-e", "PCCLI_MOCK=true",
                   "-e", "PCCLI_CLIENT_ID=", "-e", "PCCLI_CLIENT_SECRET=",
                   "-e", "PCCLI_MOCK_FIXTURE=/vhs/testdata/mocks/services-library.json",
                   "-e", "VHS_NO_SANDBOX=true",
                   "-e", "PATH=/vhs/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
                   self.settings.vhs_image]
        command += ["-c", "mkdir -p \"$HOME\" && sh scripts/record-demos.sh --binary /vhs/bin/pccli"]
        run(command)
        self.build_jekyll(source, destination, head, dataset, f"{self.settings.base_path}/versions/{head['major']}")

    def build_global(self, source, destination, head, dataset, snapshot):
        # Release metadata is global. Older tagged documentation predates these
        # pages, so use today's publisher templates only for missing global pages.
        # Frozen major snapshots continue to contain exactly their tagged pages.
        for name in ("releases.md", "downloads.md"):
            target = source / "docs" / name
            if target.is_file():
                continue
            fallback = self.settings.repository_root / "docs" / name
            if target.exists() or not fallback.is_file() or fallback.is_symlink():
                raise PublishError(f"global {name} is absent from both the released docs and publisher templates")
            if not fallback.resolve().is_relative_to(self.settings.repository_root.resolve()):
                raise PublishError(f"global metadata template {name} escapes the publisher repository")
            shutil.copy2(fallback, target)
        assets = snapshot / "assets"
        if assets.is_dir():
            for asset in assets.rglob("*"):
                if asset.is_file() and asset.suffix.lower() in MEDIA_SUFFIXES:
                    target = source / "docs" / "assets" / asset.relative_to(assets)
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(asset, target)
        self.build_jekyll(source, destination, head, dataset, self.settings.base_path)

    def build_jekyll(self, source, destination, head, dataset, base):
        ensure_search_metadata(source)
        data = source / "docs" / "_data"
        data.mkdir(exist_ok=True)
        write_json(data / "releases.json", dataset)
        overlay = source / "publish-overlay.yml"
        write_json(overlay, {"baseurl": base, "docs_root": self.settings.base_path,
                             "release_version": head["tag"], "release_major": head["major"]})
        destination.mkdir()
        # The pinned Pages image owns its Gemfile in its default working directory.
        # Do not change that directory or attempt an online bundle installation.
        run(["docker", "run", "--rm", "--network", "none", "--user", f"{os.getuid()}:{os.getgid()}",
             "--entrypoint", "bundle", "-e", "HOME=/tmp", "-v", f"{source}:/site", "-v", f"{destination}:/output",
             self.settings.jekyll_image, "exec", "jekyll", "build", "--source", "/site/docs",
             "--destination", "/output", "--config", "/site/docs/_config.yml,/site/publish-overlay.yml"])


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")


def promote(complete, state):
    """Replace the public tree with rollback, retaining the exact .git marker."""
    state.parent.mkdir(parents=True, exist_ok=True)
    transaction = Path(tempfile.mkdtemp(prefix=".pccli-docs-promote-", dir=state.parent))
    previous, replacement = transaction / "previous", transaction / "replacement"
    existed, installed = state.exists(), False
    try:
        # Copy first, so differing filesystem devices never weaken rollback.
        shutil.copytree(complete, replacement)
        if existed:
            os.replace(state, previous)
        try:
            os.replace(replacement, state)
            installed = True
            marker = previous / ".git"
            if existed and marker.exists():
                os.replace(marker, state / ".git")
        except BaseException:
            if installed:
                if existed and (state / ".git").exists() and not (previous / ".git").exists():
                    os.replace(state / ".git", previous / ".git")
                shutil.rmtree(state)
            if existed:
                os.replace(previous, state)
            raise
    finally:
        shutil.rmtree(transaction)


def publish(settings, builder=None):
    base = normalize_base(settings.base_path)
    if Path(settings.state_dir).is_symlink() or Path(settings.work_dir).is_symlink():
        raise PublishError("state and work directories cannot be symbolic links")
    repo, state, work = (Path(path).resolve() for path in
                         (settings.repository_root, settings.state_dir, settings.work_dir))
    if repo == state or state == work or state in work.parents or work in state.parents:
        raise PublishError("repository, state, and work directories must not overlap unsafely")
    for image in (settings.vhs_image, settings.jekyll_image):
        if not image or image.startswith("-") or any(character.isspace() for character in image):
            raise PublishError("build images must be explicit Docker image references")
    releases = json.loads(Path(settings.releases_json).read_text())
    dataset = release_plan(releases, settings.latest_tag, base)
    saved = read_state(state)
    # Keep identities even after a patch replaces its major head, so retargeting
    # any previously published source tag cannot pass unnoticed on a later run.
    known = dict(saved["tags"])
    for tag, old_commit in known.items():
        if commit_for(repo, tag) != old_commit:
            raise PublishError(f"previously published tag {tag} was retargeted; publication refused")
    heads = []
    for head in dataset["majors"]:
        commit = commit_for(repo, head["tag"])
        known[head["tag"]] = commit
        heads.append({**head, "commit": commit})
    previous_heads = {head["major"]: head for head in saved["majors"]}
    for head in heads:
        old = previous_heads.get(head["major"])
        if old and old["docs_url"] != head["docs_url"]:
            raise PublishError("moving the hosting base requires an explicit archive migration; "
                               f"saved {head['major']} URL is {old['docs_url']!r}, expected {head['docs_url']!r}")
    removed = set(previous_heads) - {head["major"] for head in heads}
    if removed:
        raise PublishError("release history omitted previously published major(s): " + ", ".join(sorted(removed)))
    work.mkdir(parents=True, exist_ok=True)
    builder = builder or Builder(settings)
    with tempfile.TemporaryDirectory(prefix="pccli-docs-build-", dir=work) as temporary:
        workspace = Path(temporary)
        complete = workspace / "complete"
        (complete / "versions").mkdir(parents=True)
        for head in heads:
            old = previous_heads.get(head["major"])
            output = complete / "versions" / head["major"]
            if old and old["tag"] == head["tag"] and old["commit"] == head["commit"]:
                shutil.copytree(state / "versions" / head["major"], output, copy_function=shutil.copy2)
                if old.get("legacy_text_only"):
                    head["legacy_text_only"] = True
            else:
                source = workspace / f"source-{head['major']}"
                archive_tag(repo, head["tag"], source, workspace, head["commit"])
                if prepare_legacy_docs(source, head):
                    head["legacy_text_only"] = True
                builder.build_major(source, output, head, dataset)
            if not (output / "index.html").is_file():
                raise PublishError(f"major build {head['tag']} did not produce index.html")
            exclude_archive_from_search(output)
        latest = next(head for head in heads if head["tag"] == settings.latest_tag)
        source = workspace / "source-global"
        archive_tag(repo, latest["tag"], source, workspace, latest["commit"])
        prepare_legacy_docs(source, latest)
        global_output = workspace / "global"
        builder.build_global(source, global_output, latest, dataset, complete / "versions" / latest["major"])
        if not (global_output / "index.html").is_file():
            raise PublishError("global latest build did not produce index.html")
        for item in global_output.iterdir():
            if item.name in {".git", "versions", "versions.json", "latest"}:
                raise PublishError(f"global build produced reserved path {item.name}")
            target = complete / item.name
            if item.is_dir():
                shutil.copytree(item, target)
            else:
                shutil.copy2(item, target)
        shutil.copytree(global_output, complete / "latest")
        metadata = {"schema": 1, "latest": settings.latest_tag,
                    "latest_major": dataset["latest_major"], "majors": heads, "tags": known}
        write_json(complete / "versions.json", metadata)
        (complete / ".nojekyll").write_text("")
        for path in complete.rglob("*"):
            if path.is_symlink():
                raise PublishError("generated publication contains a symbolic link")
        for tag, commit in known.items():
            if commit_for(repo, tag) != commit:
                raise PublishError(f"tag {tag} changed during the build; publication refused")
        promote(complete, state)
    return metadata


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("repository-root", "state-dir", "work-dir", "releases-json"):
        parser.add_argument("--" + name, required=True, type=Path)
    for name in ("base-path", "latest-tag", "vhs-image", "jekyll-image"):
        parser.add_argument("--" + name, required=True)
    arguments = parser.parse_args(argv)
    try:
        publish(Settings(**vars(arguments)))
    except (PublishError, OSError, subprocess.SubprocessError, json.JSONDecodeError) as error:
        parser.exit(1, f"Documentation publication failed: {error}\n")


if __name__ == "__main__":
    main()
