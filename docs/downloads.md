---
title: Downloads
nav_order: 9
---

# Download pccli

pccli is an unofficial Planning Center command line tool, not built, endorsed, or supported by Planning Center.

Choose the archive for your operating system and CPU. Extract it and put `pccli` (or `pccli.exe`) in a directory on your `PATH`. See [installation]({{ '/installation.html' | relative_url }}) for setup, or [release history]({{ site.docs_root | append: '/releases.html' }}) for older versions and their files.

{% assign latest_release = site.data.releases.entries | where: 'is_latest', true | first %}
{% if latest_release %}
## Latest release: {{ latest_release.tag }}

[Release notes and all assets]({{ latest_release.url }})

{% if latest_release.downloads and latest_release.downloads.size > 0 %}
<table class="download-table">
  <thead><tr><th scope="col">Operating system</th><th scope="col">CPU</th><th scope="col">Download</th></tr></thead>
  <tbody>
  {% for download in latest_release.downloads %}
    <tr>
      <td>{% case download.os %}{% when 'darwin' %}macOS{% when 'windows' %}Windows{% when 'linux' %}Linux{% else %}{{ download.os | escape }}{% endcase %}</td>
      <td>{% case download.arch %}{% when 'amd64' %}Intel/AMD 64-bit (amd64){% when 'arm64' %}{% if download.os == 'darwin' %}Apple Silicon (ARM64){% else %}ARM64 (arm64){% endif %}{% else %}{{ download.arch | escape }}{% endcase %}</td>
      <td><a href="{{ download.url | escape }}">{{ download.name | escape }}</a><br><small>{% if download.name contains '.zip' %}ZIP archive{% else %}tar.gz archive{% endif %}</small></td>
    </tr>
  {% endfor %}
  </tbody>
</table>

{% if latest_release.checksum_url %}
### Verify your download

[Download the SHA-256 checksums]({{ latest_release.checksum_url }}). Calculate your downloaded archive's SHA-256 and compare it with the line bearing the same filename in that file before extracting it.
{% else %}
A checksum file is not currently listed for this release. See the [GitHub release]({{ latest_release.url }}) for available assets.
{% endif %}
{% else %}
No downloadable archives are currently listed for this release. Check the [GitHub release]({{ latest_release.url }}) for published assets, or [install with Go]({{ '/installation.html' | relative_url }}#install-with-go).
{% endif %}
{% else %}
Current downloads are available on [GitHub Releases](https://github.com/joe--cool/pccli/releases). Open the latest release to choose an archive for your operating system and CPU; only published assets are downloadable.
{% endif %}

Newer Intel/AMD PCs generally use `amd64`; Apple Silicon Macs and other ARM64 systems use `arm64`. Choose based on the machine where pccli will run.
