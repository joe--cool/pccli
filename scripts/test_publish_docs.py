"""Release publication tests use local Git tags and an injected offline builder."""

from dataclasses import replace
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import publish_docs as publisher


def git(repo, *arguments):
    return subprocess.run(["git", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false",
                           "-c", "user.name=Docs tests", "-c", "user.email=docs@example.test",
                           "-C", str(repo), *arguments], check=True, text=True,
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout.strip()


def release(tag, *, prerelease=False, draft=False, missing=()):
    version = tag.removeprefix("pccli-").removeprefix("v")
    assets = []
    for operating_system, architecture, _ in publisher.PLATFORMS:
        extension = "zip" if operating_system == "windows" else "tar.gz"
        name = f"pccli_{version}_{operating_system}_{architecture}.{extension}"
        if name not in missing:
            assets.append({"name": name, "size": 42, "browser_download_url": f"https://github.com/example/pccli/releases/download/{tag}/{name}"})
    assets.append({"name": "checksums.txt", "size": 123, "browser_download_url": f"https://github.com/example/pccli/releases/download/{tag}/checksums.txt"})
    return {"tag_name": tag, "name": f"Release {tag}", "body": f"Notes for {tag}\n\n- Actual release changes.",
            "html_url": f"https://github.com/example/pccli/releases/tag/{tag}",
            "published_at": f"2026-01-{min(28, len(tag)):02d}T12:00:00Z", "assets": assets,
            "draft": draft, "prerelease": prerelease}


def tree_bytes(directory):
    return {str(path.relative_to(directory)): path.read_bytes()
            for path in directory.rglob("*") if path.is_file()}


class OfflineBuilder(publisher.Builder):
    def __init__(self, settings):
        super().__init__(settings)
        self.major_calls = []
        self.global_calls = []
        self.sources = {}
        self.fail_global = False

    def build_major(self, source, destination, head, dataset):
        self.major_calls.append(head["tag"])
        self.sources[head["tag"]] = {str(path.relative_to(source)) for path in source.rglob("*")}
        if not head.get("legacy_text_only"):
            assets = source / "docs" / "assets"
            assets.mkdir(exist_ok=True)
            (assets / "demo.mp4").write_bytes(f"video for {head['tag']}".encode())
        self.build_jekyll(source, destination, head, dataset,
                          f"{self.settings.base_path}/versions/{head['major']}")

    def build_global(self, source, destination, head, dataset, snapshot):
        self.global_calls.append(head["tag"])
        if self.fail_global:
            raise publisher.PublishError("simulated late Jekyll failure")
        super().build_global(source, destination, head, dataset, snapshot)

    def build_jekyll(self, source, destination, head, dataset, base):
        destination.mkdir()
        marker = (source / "docs" / "index.md").read_text()
        (destination / "index.html").write_text(f"<html><head><title>Docs</title></head><body>{marker}\n{head['tag']}\n{base}\n</body></html>")
        publisher.write_json(destination / "releases.json", dataset)
        for page in (source / "docs").glob("*.md"):
            if page.name != "index.md":
                (destination / page.with_suffix(".html").name).write_text("<html><head></head><body>" + page.read_text() + "</body></html>")
        assets = source / "docs" / "assets"
        if assets.exists():
            import shutil
            shutil.copytree(assets, destination / "assets")


class PublicationTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.repo = self.root / "repository"
        self.repo.mkdir()
        git(self.repo, "init", "--quiet")
        (self.repo / "docs").mkdir()
        (self.repo / "scripts").mkdir()
        (self.repo / "docs" / "_config.yml").write_text("title: pccli\n")
        (self.repo / "scripts" / "demo.tape").write_text("Output docs/assets/demo.mp4\n")
        self.state = self.root / "state"
        self.state.mkdir()
        (self.state / ".git").write_bytes(b"gitdir: preserve this worktree marker\n")
        self.releases_file = self.root / "releases.json"
        self.settings = publisher.Settings(self.repo, self.state, self.root / "work", "/pccli",
                                             self.releases_file, "v2.0.0", "vhs@sha256:synthetic", "jekyll@sha256:synthetic")
        self.tag("pccli-v1.0.0", "legacy exact source")
        self.tag("v2.0.0", "current exact source")
        # These are deliberately absent from git archive, even when the live
        # checkout differs from the tag being rendered.
        (self.repo / ".env").write_text("DO_NOT_INCLUDE=synthetic-untracked-sentinel\n")
        (self.repo / "untracked.txt").write_text("not released")
        (self.repo / "docs" / "index.md").write_text("uncommitted working copy")
        (self.repo / "docs/releases.md").write_text("Current global release history template")
        (self.repo / "docs/downloads.md").write_text("Current global download metadata template")
        self.releases = [release("pccli-v1.0.0"), release("v2.0.0")]
        self.write_releases()

    def tag(self, tag, marker):
        if not tag.startswith("pccli-v1."):
            (self.repo / "scripts/record-demos.sh").write_text("#!/bin/sh\n# Synthetic read-only recorder.\n")
        (self.repo / "docs" / "index.md").write_text(marker)
        git(self.repo, "add", "docs", "scripts")
        git(self.repo, "commit", "--quiet", "-m", marker)
        git(self.repo, "tag", tag)

    def write_releases(self):
        publisher.write_json(self.releases_file, self.releases)

    def publish(self, settings=None, builder=None):
        settings = settings or self.settings
        builder = builder or OfflineBuilder(settings)
        publisher.publish(settings, builder)
        return builder

    def test_archive_search_policy_migrates_saved_html_without_rebuilding(self):
        self.publish()
        saved = self.state / "versions/v1/index.html"
        text = saved.read_text().replace('<meta name="robots" content="noindex, follow">', '')
        text = text.replace('</head>', '<link href="https://example.test/old" rel="canonical"><meta content="index" name="robots"></head>')
        saved.write_text(text)
        video = (self.state / "versions/v2/assets/demo.mp4").read_bytes()
        builder = self.publish()
        self.assertEqual(builder.major_calls, [])
        for major in ("v1", "v2"):
            text = (self.state / "versions" / major / "index.html").read_text()
            self.assertEqual(text.count('name="robots"'), 1)
            self.assertIn('content="noindex, follow"', text)
            self.assertNotIn('rel="canonical"', text)
        self.assertNotIn('noindex', (self.state / "index.html").read_text())
        self.assertNotIn('noindex', (self.state / "latest/index.html").read_text())
        self.assertEqual(video, (self.state / "versions/v2/assets/demo.mp4").read_bytes())
        before = tree_bytes(self.state / "versions")
        self.publish()
        self.assertEqual(before, tree_bytes(self.state / "versions"))

    def test_canonical_metadata_and_malformed_archive_fail_safely(self):
        layout = self.repo / "docs/_layouts/default.html"
        layout.parent.mkdir()
        layout.write_text('<html><head><title>Docs</title></head><body></body></html>')
        publisher.ensure_search_metadata(self.repo)
        text = layout.read_text()
        self.assertIn('site.url | append: site.docs_root | append: page.url', text)
        self.assertIn("replace: '/index.html', '/'", text)
        publisher.ensure_search_metadata(self.repo)
        self.assertEqual(text, layout.read_text())
        self.publish()
        (self.state / "versions/v1/index.html").write_text('<html><body>No head</body></html>')
        before = tree_bytes(self.state)
        with self.assertRaisesRegex(publisher.PublishError, 'no head for indexing policy'):
            self.publish()
        self.assertEqual(before, tree_bytes(self.state))

    def test_two_majors_exact_tag_sources_and_latest_aliases(self):
        builder = self.publish()
        self.assertCountEqual(builder.major_calls, ["pccli-v1.0.0", "v2.0.0"])
        self.assertIn("legacy exact source", (self.state / "versions/v1/index.html").read_text())
        self.assertIn("current exact source", (self.state / "index.html").read_text())
        self.assertEqual((self.state / "index.html").read_bytes(), (self.state / "latest/index.html").read_bytes())
        self.assertEqual((self.state / "assets/demo.mp4").read_bytes(), b"video for v2.0.0")
        self.assertEqual((self.state / ".git").read_bytes(), b"gitdir: preserve this worktree marker\n")
        for files in builder.sources.values():
            self.assertNotIn(".env", files)
            self.assertNotIn("untracked.txt", files)
        self.assertEqual(json.loads((self.state / "versions.json").read_text())["latest"], "v2.0.0")
        self.assertEqual(list((self.root / "work").iterdir()), [])

    def test_old_tag_uses_global_metadata_templates_without_changing_frozen_major(self):
        builder = self.publish()
        self.assertNotIn("docs/releases.md", builder.sources["v2.0.0"])
        self.assertNotIn("docs/downloads.md", builder.sources["v2.0.0"])
        self.assertFalse((self.state / "versions/v2/releases.html").exists())
        self.assertFalse((self.state / "versions/v2/downloads.html").exists())
        self.assertIn("Current global release history template", (self.state / "releases.html").read_text())
        self.assertIn("Current global download metadata template", (self.state / "downloads.html").read_text())
        self.assertEqual((self.state / "latest/releases.html").read_bytes(), (self.state / "releases.html").read_bytes())
        before = tree_bytes(self.state)
        (self.repo / "docs/releases.md").unlink()
        with self.assertRaisesRegex(publisher.PublishError, "absent from both"):
            self.publish()
        self.assertEqual(before, tree_bytes(self.state))

    def test_patch_replaces_only_its_major_and_old_snapshot_is_identical(self):
        self.publish()
        old_v1 = tree_bytes(self.state / "versions/v1")
        old_v2 = tree_bytes(self.state / "versions/v2")
        self.tag("v2.0.1", "patched exact source")
        self.releases.append(release("v2.0.1"))
        self.write_releases()
        settings = replace(self.settings, latest_tag="v2.0.1")
        builder = self.publish(settings)
        self.assertEqual(builder.major_calls, ["v2.0.1"])
        self.assertEqual(old_v1, tree_bytes(self.state / "versions/v1"))
        self.assertNotEqual(old_v2, tree_bytes(self.state / "versions/v2"))
        dataset = json.loads((self.state / "releases.json").read_text())
        self.assertEqual({entry["tag"] for entry in dataset["entries"]}, {"pccli-v1.0.0", "v2.0.0", "v2.0.1"})
        metadata = json.loads((self.state / "versions.json").read_text())
        self.assertIn("v2.0.0", metadata["tags"])
        self.assertEqual(metadata["latest_major"], "v2")

    def test_release_metadata_refresh_rebuilds_global_only_and_reuses_video(self):
        self.publish()
        snapshots = tree_bytes(self.state / "versions")
        self.releases.append(release("v3.0.0-rc.1", prerelease=True))
        self.releases.append(release("v3.0.0", draft=True))
        self.releases[1]["body"] = "Edited published release notes"
        self.write_releases()
        builder = self.publish()
        self.assertEqual(builder.major_calls, [])
        self.assertEqual(builder.global_calls, ["v2.0.0"])
        self.assertEqual(snapshots, tree_bytes(self.state / "versions"))
        dataset = json.loads((self.state / "releases.json").read_text())
        self.assertEqual(len(dataset["entries"]), 3)
        prerelease = next(entry for entry in dataset["entries"] if entry["prerelease"])
        self.assertEqual(prerelease["tag"], "v3.0.0-rc.1")
        self.assertIsNone(prerelease["docs_url"])
        self.assertEqual((self.state / "assets/demo.mp4").read_bytes(), b"video for v2.0.0")

    def test_actual_asset_urls_notes_and_absent_files_are_not_invented(self):
        missing = "pccli_2.0.0_windows_arm64.zip"
        self.releases[1] = release("v2.0.0", missing=(missing,))
        self.write_releases()
        self.publish()
        dataset = json.loads((self.state / "releases.json").read_text())
        current = next(entry for entry in dataset["entries"] if entry["is_latest"])
        self.assertEqual(len(current["downloads"]), 5)
        self.assertNotIn(missing, {asset["name"] for asset in current["downloads"]})
        actual = {asset["browser_download_url"] for asset in self.releases[1]["assets"]}
        self.assertTrue({asset["url"] for asset in current["downloads"]} <= actual)
        self.assertIn(current["checksum_url"], actual)
        self.assertEqual(current["notes"], self.releases[1]["body"])
        legacy = next(entry for entry in dataset["entries"] if entry["tag"] == "pccli-v1.0.0")
        self.assertEqual(len(legacy["downloads"]), 6)
        self.assertEqual(legacy["docs_url"], "/pccli/versions/v1/")
        self.assertEqual(current["docs_url"], "/pccli/versions/v2/")

    def test_late_failure_leaves_all_state_unchanged(self):
        self.publish()
        before = tree_bytes(self.state)
        self.tag("v2.1.0", "minor release source")
        self.releases.append(release("v2.1.0"))
        self.write_releases()
        settings = replace(self.settings, latest_tag="v2.1.0")
        builder = OfflineBuilder(settings)
        builder.fail_global = True
        with self.assertRaisesRegex(publisher.PublishError, "late Jekyll"):
            self.publish(settings, builder)
        self.assertEqual(before, tree_bytes(self.state))
        self.assertEqual(list((self.root / "work").iterdir()), [])

    def test_hosting_base_migration_is_explicit_and_preserves_saved_snapshots(self):
        self.publish()
        before = tree_bytes(self.state)
        settings = replace(self.settings, base_path="/different-hosting-base")
        builder = OfflineBuilder(settings)
        with self.assertRaisesRegex(publisher.PublishError, "explicit archive migration"):
            self.publish(settings, builder)
        self.assertEqual(builder.major_calls, [])
        self.assertEqual(builder.global_calls, [])
        self.assertEqual(before, tree_bytes(self.state))
        metadata = json.loads((self.state / "versions.json").read_text())
        del metadata["majors"][0]["docs_url"]
        publisher.write_json(self.state / "versions.json", metadata)
        invalid_before = tree_bytes(self.state)
        with self.assertRaisesRegex(publisher.PublishError, "missing major documentation URL"):
            self.publish()
        self.assertEqual(invalid_before, tree_bytes(self.state))

    def test_retargeted_head_and_superseded_tags_are_rejected(self):
        self.publish()
        self.tag("v2.0.1", "patch source")
        self.releases.append(release("v2.0.1"))
        self.write_releases()
        settings = replace(self.settings, latest_tag="v2.0.1")
        self.publish(settings)
        before = tree_bytes(self.state)
        git(self.repo, "tag", "-f", "v2.0.0", "v2.0.1")
        builder = OfflineBuilder(settings)
        with self.assertRaisesRegex(publisher.PublishError, "retargeted"):
            self.publish(settings, builder)
        self.assertEqual(builder.major_calls, [])
        self.assertEqual(before, tree_bytes(self.state))

    def test_unknown_state_latest_mismatch_and_unsafe_inputs_fail_before_build(self):
        (self.state / "unmanaged.txt").write_text("preserve")
        with self.assertRaisesRegex(publisher.PublishError, "unknown files"):
            self.publish()
        (self.state / "unmanaged.txt").unlink()
        self.tag("v2.0.1", "next patch")
        self.releases.append(release("v2.0.1"))
        self.write_releases()
        with self.assertRaisesRegex(publisher.PublishError, "latest tag must match"):
            self.publish()
        for base in ("../x", "/pccli/../x", "/pccli/", "//host"):
            with self.assertRaises(publisher.PublishError):
                publisher.normalize_base(base)
        for tag in ("../../x", "--help", "v2.0.0/path", "v2.0.0\n"):
            with self.assertRaises(publisher.PublishError):
                publisher.tag_version(tag)

    def test_empty_base_and_canonical_tag_preference(self):
        git(self.repo, "tag", "pccli-v2.0.0", "v2.0.0")
        self.releases.append(release("pccli-v2.0.0"))
        self.write_releases()
        self.publish(replace(self.settings, base_path=""))
        dataset = json.loads((self.state / "releases.json").read_text())
        self.assertEqual(dataset["majors"][0]["tag"], "v2.0.0")
        self.assertEqual(dataset["majors"][0]["docs_url"], "/versions/v2/")

    def test_tracked_dotenv_and_symlinks_are_rejected(self):
        git(self.repo, "add", ".env")
        git(self.repo, "commit", "--quiet", "-m", "synthetic forbidden dotenv")
        git(self.repo, "tag", "v2.0.1")
        self.releases.append(release("v2.0.1"))
        self.write_releases()
        with self.assertRaisesRegex(publisher.PublishError, "tracked .env"):
            self.publish(replace(self.settings, latest_tag="v2.0.1"))
        self.assertEqual(set(tree_bytes(self.state)), {".git"})

    def test_tracked_contributor_alias_is_skipped_but_other_symlinks_are_rejected(self):
        (self.repo / "AGENTS.md").write_text("Synthetic contributor instructions")
        (self.repo / "CLAUDE.md").symlink_to("AGENTS.md")
        git(self.repo, "add", "AGENTS.md", "CLAUDE.md")
        git(self.repo, "commit", "--quiet", "-m", "tracked contributor alias")
        git(self.repo, "tag", "v2.0.1")
        self.releases.append(release("v2.0.1"))
        self.write_releases()
        settings = replace(self.settings, latest_tag="v2.0.1")
        builder = self.publish(settings)
        self.assertIn("AGENTS.md", builder.sources["v2.0.1"])
        self.assertNotIn("CLAUDE.md", builder.sources["v2.0.1"])
        before = tree_bytes(self.state)
        (self.repo / "docs/unsafe-link.md").symlink_to("/etc/passwd")
        git(self.repo, "add", "docs/unsafe-link.md")
        git(self.repo, "commit", "--quiet", "-m", "synthetic unsafe link")
        git(self.repo, "tag", "v2.0.2")
        self.releases.append(release("v2.0.2"))
        self.write_releases()
        with self.assertRaisesRegex(publisher.PublishError, "unsupported link"):
            self.publish(replace(settings, latest_tag="v2.0.2"))
        self.assertEqual(before, tree_bytes(self.state))

    def test_promotion_failure_rolls_back_git_marker_and_public_tree(self):
        self.publish()
        before = tree_bytes(self.state)
        real_replace = os.replace
        calls = 0
        def fail_install(source, destination):
            nonlocal calls
            calls += 1
            if calls == 2:
                raise OSError("injected install failure")
            return real_replace(source, destination)
        with patch.object(publisher.os, "replace", side_effect=fail_install):
            with self.assertRaisesRegex(OSError, "install failure"):
                self.publish()
        self.assertEqual(before, tree_bytes(self.state))
        self.assertFalse(list(self.root.glob(".pccli-docs-promote-*")))


class BuildCommandTest(unittest.TestCase):
    def test_go_module_version_and_offline_docker_contract(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "source"
            (source / "scripts").mkdir(parents=True)
            (source / "scripts/record-demos.sh").write_text("#!/bin/sh\n")
            settings = publisher.Settings(source, root / "state", root / "work", "/pccli", root / "releases", "v2.3.4", "vhs:pinned", "jekyll:pinned")
            builder = publisher.Builder(settings)
            commands = []
            def execute(argv, **keywords):
                commands.append((argv, keywords))
                return "github.com/example/pccli/v2" if argv[:3] == ["go", "list", "-m"] else ""
            with patch.object(publisher, "run", side_effect=execute), patch.object(builder, "build_jekyll"):
                builder.build_major(source, root / "output", {"tag": "v2.3.4", "major": "v2"}, {})
            build = next((argv, options) for argv, options in commands if argv[:2] == ["go", "build"])
            self.assertIn("github.com/example/pccli/v2/internal/cli.Version=2.3.4", " ".join(build[0]))
            self.assertEqual(build[1]["env"]["CGO_ENABLED"], "0")
            docker = next(argv for argv, _ in commands if argv[0] == "docker")
            self.assertEqual(docker[docker.index("--network") + 1], "none")
            self.assertIn("--binary /vhs/bin/pccli", docker[-1])
            self.assertIn("PCCLI_MOCK=true", docker)
            self.assertIn("HOME=/tmp/pccli-docs-home", docker)

    def test_legacy_source_is_text_only_with_pinned_install_and_history_banner(self):
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary)
            (source / "docs/_layouts").mkdir(parents=True)
            (source / "docs/index.md").write_text('<img class="terminal-demo" src="library-demo.gif" />\ngo install github.com/example/pccli/cmd/pccli@latest\n')
            (source / "docs/_layouts/default.html").write_text('<html><body><nav>Existing navigation</nav></body></html>')
            head = {"tag": "pccli-v1.0.0", "major": "v1"}
            self.assertTrue(publisher.prepare_legacy_docs(source, head))
            text = (source / "docs/index.md").read_text()
            self.assertIn("The terminal recording was not archived", text)
            self.assertNotIn("<img", text)
            self.assertIn("@pccli-v1.0.0", text)
            layout = (source / "docs/_layouts/default.html").read_text()
            self.assertIn("Existing navigation", layout)
            self.assertIn("site.docs_root", layout)
            self.assertIn("release_version | escape", layout)
            settings = publisher.Settings(source, source / "state", source / "work", "/pccli", source / "releases", "pccli-v1.0.0", "vhs:pinned", "jekyll:pinned")
            builder = publisher.Builder(settings)
            with patch.object(publisher, "run") as execute, patch.object(builder, "build_jekyll") as jekyll:
                builder.build_major(source, source / "output", head, {})
            execute.assert_not_called()
            jekyll.assert_called_once()

    def test_prefix_era_v2_install_is_pinned_without_changing_recorder(self):
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary)
            (source / "docs").mkdir()
            (source / "scripts").mkdir()
            (source / "scripts/record-demos.sh").write_text("modern read-only recorder")
            (source / "docs/_layouts").mkdir()
            (source / "docs/_layouts/default.html").write_text("<html><body><nav>Original v2 navigation</nav></body></html>")
            (source / "docs/index.md").write_text("go install github.com/example/pccli/cmd/pccli@latest\ngo install github.com/example/pccli/v2/cmd/pccli@latest\n")
            self.assertFalse(publisher.prepare_legacy_docs(source, {"tag": "pccli-v2.0.0", "major": "v2"}))
            text = (source / "docs/index.md").read_text()
            self.assertIn("github.com/example/pccli/cmd/pccli@pccli-v2.0.0", text)
            self.assertIn("github.com/example/pccli/v2/cmd/pccli@latest", text)
            self.assertEqual((source / "scripts/record-demos.sh").read_text(), "modern read-only recorder")
            layout = (source / "docs/_layouts/default.html").read_text()
            self.assertIn("Original v2 navigation", layout)
            self.assertIn("site.release_version | escape", layout)
            self.assertIn("site.docs_root", layout)
            self.assertIn("/releases.html", layout)
            self.assertIn("/downloads.html", layout)

    def test_current_release_layout_is_preserved_byte_for_byte(self):
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary)
            (source / "docs/_layouts").mkdir(parents=True)
            (source / "scripts").mkdir()
            (source / "scripts/record-demos.sh").write_text("modern recorder")
            modern = '<body><nav>{{ site.docs_root }} /releases.html /downloads.html {{ site.release_version | escape }}</nav></body>\n'
            layout = source / "docs/_layouts/default.html"
            layout.write_text(modern)
            self.assertFalse(publisher.prepare_legacy_docs(source, {"tag": "v2.1.0", "major": "v2"}))
            self.assertEqual(layout.read_bytes(), modern.encode())

    def test_jekyll_overlay_and_image_working_directory(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "source"
            (source / "docs").mkdir(parents=True)
            settings = publisher.Settings(source, root / "state", root / "work", "/pccli", root / "releases", "v2.3.4", "vhs:pinned", "jekyll:pinned")
            with patch.object(publisher, "run") as execute:
                publisher.Builder(settings).build_jekyll(source, root / "output", {"tag": "v2.3.4", "major": "v2"}, {"entries": []}, "/pccli/versions/v2")
            command = execute.call_args.args[0]
            self.assertNotIn("-w", command)
            self.assertIn("bundle", command)
            self.assertEqual(command[command.index("--network") + 1], "none")
            self.assertIn("/site/docs/_config.yml,/site/publish-overlay.yml", command)
            overlay = json.loads((source / "publish-overlay.yml").read_text())
            self.assertEqual(overlay, {"baseurl": "/pccli/versions/v2", "docs_root": "/pccli", "release_version": "v2.3.4", "release_major": "v2"})
            self.assertEqual(json.loads((source / "docs/_data/releases.json").read_text()), {"entries": []})


if __name__ == "__main__":
    unittest.main()
