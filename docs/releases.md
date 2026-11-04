---
title: Releases
nav_order: 8
---

# Releases

Find what changed, download a release, and open the documentation for its major version. pccli is unofficial and is not built, endorsed, or supported by Planning Center.

[Latest release documentation]({{ site.docs_root | append: '/' }}) · [Download pccli]({{ site.docs_root | append: '/downloads.html' }}) · [All GitHub releases](https://github.com/joe--cool/pccli/releases)

Documentation follows the latest released changes within each major version. Release notes below retain the history of individual minor and patch releases. For the current release history, open [all releases]({{ site.docs_root | append: '/releases.html' }}).

{% if site.data.releases.majors and site.data.releases.majors.size > 0 %}
## Documentation by major version

{% for release_major in site.data.releases.majors %}
- [{{ release_major.major }} docs]({{ release_major.docs_url }}) — updated for {{ release_major.tag }}.
{% endfor %}
{% endif %}

{% if site.data.releases.entries and site.data.releases.entries.size > 0 %}
{% for release in site.data.releases.entries %}
<article class="release-entry">
  <header class="release-header">
    <h2 id="{{ release.tag | slugify }}">{{ release.name | default: release.tag | escape }}</h2>
    <p class="release-meta">
      <code>{{ release.tag | escape }}</code>
      {% if release.published_at %}<span aria-hidden="true"> · </span><time datetime="{{ release.published_at | escape }}">{{ release.published_at | date: '%B %-d, %Y' }}</time>{% endif %}
      {% if release.is_latest %}<span class="release-badge">Latest release</span>{% endif %}
      {% if release.prerelease %}<span class="release-badge">Prerelease</span>{% endif %}
    </p>
  </header>
  <p class="release-links">
    <a href="{{ release.url | escape }}">Release and downloads</a>
    {% if release.docs_url %}
      <span aria-hidden="true"> · </span><a href="{{ release.docs_url | escape }}">{{ release.major | escape }} docs</a>
    {% else %}
      <span aria-hidden="true"> · </span><span>Documentation snapshot unavailable.</span>
    {% endif %}
  </p>
  {% if release.downloads and release.downloads.size > 0 %}
  <details class="release-downloads">
    <summary>Downloads for {{ release.tag | escape }}</summary>
    <ul>
      {% for download in release.downloads %}
      <li><a href="{{ download.url | escape }}">{{ download.label | escape }} — {{ download.name | escape }}</a></li>
      {% endfor %}
    </ul>
    {% if release.checksum_url %}<p><a href="{{ release.checksum_url | escape }}">SHA-256 checksums</a></p>{% endif %}
  </details>
  {% endif %}
  <div class="release-notes">
    {% if release.notes and release.notes != '' %}
      {{ release.notes | markdownify }}
    {% else %}
      <p>No release notes were published. See the GitHub release for details.</p>
    {% endif %}
  </div>
</article>
{% endfor %}
{% else %}
Release history is available on [GitHub Releases](https://github.com/joe--cool/pccli/releases), including published notes and downloads.
{% endif %}
