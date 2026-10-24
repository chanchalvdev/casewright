# Deploying to Oracle Cloud (Always Free)

One Ampere A1 VM (arm64, 4 OCPU / 24 GB) runs the whole stack at no cost:

```
internet ──► :443 Caddy (Let's Encrypt) ──► web ──► api
             internal network only: postgres · qdrant · ollama · phoenix
```

- **Only Caddy is public.** Postgres, Qdrant, Ollama and the API have no host
  ports. Phoenix (live traces) listens on the VM's loopback only.
- **Production classifies with Ollama** (`llama3.2:3b`), so real logs never
  reach a cloud LLM. `docker-compose.yml` forces `LLM_PROVIDER=ollama`.
- **Releases deploy themselves** after you approve them: pushing a tag
  `vX.Y.Z` builds the images, then the `deploy` job in `release.yml` waits for
  approval, copies these files to the VM and runs `deploy.sh`.
- **A failed release rolls back.** `deploy.sh` waits for `/health/ready`
  (LLM, Postgres and Qdrant answering) and restarts the previous tag if it
  does not get there.

Expect **15–30 s per LLM classification** on CPU. Rules-first mode means
events a detection rule matches never wait for the LLM.

## 1. Create the VM (Oracle console, once)

1. **Compute → Instances → Create instance**
   - Image: *Canonical Ubuntu 24.04* (aarch64)
   - Shape: *Ampere VM.Standard.A1.Flex*, 4 OCPU, 24 GB
   - Boot volume: 100 GB (Always Free covers 200 GB in total)
   - SSH key: your own public key, for the `ubuntu` admin user
   - **Advanced options → Management → Cloud-init script**: paste
     `cloud-init.yaml`, after replacing the `ssh-ed25519 AAAA...` line with
     the public half of the deploy key from step 3.
2. "Out of host capacity" is common on free accounts. Try another
   availability domain or retry later. Upgrading to Pay As You Go keeps
   Always Free resources free and usually helps, but it needs a card on file.
3. **Deploy key** (on your laptop):
   `ssh-keygen -t ed25519 -f casewright-deploy -C casewright-deploy -N ''`.
   The public half goes into cloud-init; the private half becomes the
   `OCI_SSH_KEY` GitHub secret.
4. **Networking → the instance's subnet → Security list → Ingress rules**:
   - TCP 22 from **your IP only** (`x.x.x.x/32`)
   - TCP 80 and TCP 443 from `0.0.0.0/0`
   - UDP 443 from `0.0.0.0/0` (HTTP/3, optional)
   - nothing else
5. **Reserve the public IP** (instance → attached VNIC → IPv4 addresses →
   edit → reserved), so it survives a stop/start.

Cloud-init also opens 80/443 in the VM's own iptables. Oracle's Ubuntu
images reject everything except SSH there, even when the security list allows
the port. If the site is unreachable, check this first:
`sudo iptables -L INPUT --line-numbers`.

## 2. Prepare the VM (once)

```bash
ssh deploy@<public-ip>          # cloud-init created this user
cd casewright                   # created by cloud-init; the deploy job fills it
```

Create `.env` from `prod.env.example` (the first deploy job copies the
example over; you can also `scp` it yourself), then fill it in:

```bash
cp prod.env.example .env && chmod 600 .env && nano .env
```

- `DOMAIN`: your domain, or `<ip-with-dashes>.sslip.io`, e.g.
  `141-148-1-2.sslip.io`. sslip.io resolves to that IP, and Caddy gets a
  real certificate for it.
- `POSTGRES_PASSWORD`, `CASEWRIGHT_ADMIN_PASSWORD`, `CASEWRIGHT_API_KEYS`:
  `openssl rand -hex 24` each.

If the GHCR images are private, log in once with a token that has
`read:packages`: `echo <token> | docker login ghcr.io -u <github-user> --password-stdin`.
Making the two packages public on GitHub avoids that.

## 3. GitHub settings (once)

**Settings → Environments → New environment `production`**:
- Required reviewers: you. Every deploy then waits for your click.
- Deployment branches and tags: tags matching `v*`.
- Environment secrets:

| Secret | Value |
|---|---|
| `OCI_HOST` | the VM's public IP |
| `OCI_SSH_KEY` | private half of the deploy key |
| `OCI_SSH_KNOWN_HOSTS` | output of `ssh-keyscan -t ed25519 <public-ip>` (check the fingerprint against the console's serial output) |
| `OCI_API_KEY` | one of the keys in `CASEWRIGHT_API_KEYS`, for the smoke test |

- Environment variable `OCI_DOMAIN`: the value of `DOMAIN`. The deploy job
  only runs when it is set.

## 4. Release

```bash
git tag -a v1.1.0 -m v1.1.0 && git push origin v1.1.0
```

The Release workflow runs the quality gate, builds and signs the images,
then waits for your approval in the Actions tab. After approval it deploys,
checks `/health/ready` and classifies one line through the public URL.

Deploy by hand on the VM: `./deploy.sh v1.1.0`. To roll back, deploy the
previous tag.

## Operating it

| Task | Command (on the VM, in `~/casewright`) |
|---|---|
| status | `IMAGE_TAG=$(cat .deployed_tag \| sed 's/^v//') docker compose --env-file .env ps` |
| API logs | `docker logs -f casewright-api-1` |
| live traces (Phoenix) | on your laptop: `ssh -L 6006:localhost:6006 deploy@<ip>`, open http://localhost:6006 |
| model in use | `docker exec casewright-ollama-1 ollama list` |
| database backup | `docker exec casewright-postgres-1 pg_dump -U casewright casewright \| gzip > backup.sql.gz` |

Model drift: `deploy.sh` pulls a model only when it is missing, so the model
behind `llama3.2:3b` never changes under a running deployment. `ollama list`
records its digest.

## Plan B: no VM available

Run the same stack on your own machine and publish it with a free Cloudflare
quick tunnel (no account, random `*.trycloudflare.com` URL):

```bash
cd casewright/deploy/oci
cp prod.env.example .env   # DOMAIN=localhost, other values as above
PULL=0 ./deploy.sh v1.1.0  # after building or pulling the images locally
cloudflared tunnel --url https://localhost --no-tls-verify
```

`PULL=0` skips the registry and uses images already on the machine. This is
how `deploy.sh` was rehearsed, including a deliberately broken release that
rolled back.
