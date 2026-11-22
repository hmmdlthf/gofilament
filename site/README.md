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

### One-time setup

1. Create the static app on goplesk (from a machine that can reach the control
   plane; `$KEY` is `control_plane.cluster_key` in `/etc/goplesk/config.yaml`):

   ```bash
   curl -H "Authorization: Bearer $KEY" \
     -d '{"name":"gofilament-site","domain":"gofilament.elridhma.com","is_static":true}' \
     https://<control-plane>:8443/api/v1/apps/deploy
   ```

   goplesk requests the TLS certificate once the domain's DNS points at the server.

2. Add to the repository (*Settings → Secrets and variables → Actions*, or as
   secrets of the `website` environment, which the workflow creates on first run):

   | Name | Kind | Value |
   | --- | --- | --- |
   | `GOPLESK_URL` | secret | control plane URL, e.g. `https://<control-plane>:8443` |
   | `GOPLESK_CLUSTER_KEY` | secret | `control_plane.cluster_key` |
   | `GOPLESK_APP` | variable (optional) | app name, default `gofilament-site` |

3. Run *Website → Run workflow* once to publish the current release.
