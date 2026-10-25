# Live demo

A public, single-container demo of Casewright, deployed on Render's free plan
by [`render.yaml`](../../../render.yaml) at the repository root.

- Built from the released `casewright-api` and `casewright-web` images
  (`ARG CASEWRIGHT_VERSION` in the `Dockerfile`), with nginx in front.
- Rules-only (`LLM_PROVIDER=none`), in-memory storage, seeded with
  `demo/attack-scenario.log` on every start.
- Visitors sign in as the analyst `demo` / `casewright-demo`. The admin
  password and API key are random per start and never shown.
- Approved containment and notify actions go to a stand-in endpoint inside
  the container (`127.0.0.1:9000`), so nothing outside is touched.
- Resets every `DEMO_RESET_MINUTES` (default 360) by restarting the API.

Run it locally:

```bash
docker build -f casewright/deploy/live-demo/Dockerfile -t casewright-demo casewright
docker run --rm -e CASEWRIGHT_COOKIE_SECURE=false -p 7860:7860 casewright-demo
# http://localhost:7860, sign in as demo / casewright-demo
```

To move the demo to a new release, change `CASEWRIGHT_VERSION` and push.
