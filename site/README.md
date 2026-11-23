# gofilament.elridhma.com

The project website: landing page, documentation for every release, and the
changelog. Built with [Hugo](https://gohugo.io) and deployed to the goplesk
server by [`.github/workflows/site.yml`](../.github/workflows/site.yml) each time a
GitHub release is published.

## How the docs are versioned

Nothing under `content/docs/` is written by hand. `tools/gendocs` reads the
release tags and, for the newest patch of every minor release (`v1.39.1` for
`v1.39`), takes these files **as they were at that tag** (`git show <tag>:<file>`):

| File | Becomes |
| --- | --- |
| `README.md` | the version's home page (text before the first `##`) and one **Guide** page per `##` section |
| `DOCUMENTATION.md` | one **Reference** page per `##` section (from v1.26, where the file starts) |
| `CHANGELOG.md` (newest tag) | `/changelog/`, with a link from each release to its docs |

The newest release is served twice: at `/docs/latest/` (the default, and the
URL search engines index) and at `/docs/v1.40/`, which takes over when v1.41
ships. Older versions get a banner pointing to the latest docs and are kept out
of search results. Links between the files (`#anchors`, `DOCUMENTATION.md#...`)
are rewritten to the page that holds the heading; links to other repository
files go to GitHub at that tag.

So **writing docs is unchanged**: edit `README.md` / `DOCUMENTATION.md` /
`CHANGELOG.md` as today; the next release publishes them as a new version.
Which files are used, and from which version on, is set in `gendocs.json`.

The landing page text is in `data/landing.yaml`; site settings and the menu in
`hugo.toml`. The theme (`themes/godocs`) has no project-specific content, so
the golaravel site uses the same one.

## Local preview

Needs Go and [Hugo](https://gohugo.io/installation/) 0.168 or newer.

```bash
./site/build.sh server     # http://localhost:1313
./site/build.sh            # production build into site/public
```

## Deploying

Deploys run on **release published** (and by hand from the Actions tab,
*Website → Run workflow*). The job builds the site, packs `site/public` as a
`.tar.gz`, and uploads it with `POST /api/v1/apps/<app>/release`; goplesk
switches the static site to the new directory atomically.

The first deploy also creates the app: when goplesk answers `404` (no such
app), the job uploads the archive to `POST /api/v1/artifacts` and creates the
static app for the domain with `POST /api/v1/apps/deploy`. goplesk then
requests the TLS certificate, since the domain's DNS points at the server; the
job waits up to 3 minutes for `https://<domain>/docs/latest/` to answer.

If you ever create releases from another workflow using `GITHUB_TOKEN`, those
releases don't trigger this one. Call it from that workflow instead:

```yaml
jobs:
  website:
    uses: ./.github/workflows/site.yml
    secrets: inherit
```

### One-time setup

1. Add the cluster key to the repository (*Settings → Secrets and variables →
   Actions*, or as a secret of the `website` environment, which the workflow
   creates on first run). The rest have defaults:

   | Name | Kind | Value |
   | --- | --- | --- |
   | `GOPLESK_CLUSTER_KEY` | secret | `control_plane.cluster_key` from `/etc/goplesk/config.yaml` |
   | `GOPLESK_URL` | variable (optional) | control plane, default `https://server1.goplesk.caaqit.com` |
   | `GOPLESK_APP` | variable (optional) | app name, default `gofilament-site` |
   | `SITE_DOMAIN` | variable (optional) | domain, default `gofilament.elridhma.com` |

2. Run *Website → Run workflow* once. It creates the app and publishes the
   current release; later releases deploy by themselves.
