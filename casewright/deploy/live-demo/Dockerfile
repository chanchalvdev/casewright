# syntax=docker/dockerfile:1.7
# Public live demo: one container with the released API binary and dashboard,
# nginx in front, rules-only, in-memory storage. Build context: casewright/
# (render.yaml at the repository root deploys it on Render's free plan).
#   docker build -f deploy/live-demo/Dockerfile -t casewright-demo casewright

ARG CASEWRIGHT_VERSION=2.0.0
FROM ghcr.io/chanchalvdev/casewright-api:${CASEWRIGHT_VERSION} AS api
FROM ghcr.io/chanchalvdev/casewright-web:${CASEWRIGHT_VERSION} AS web

FROM nginxinc/nginx-unprivileged:1.31-alpine
USER root
RUN apk add --no-cache curl
COPY --from=api /usr/local/bin/casewright /usr/local/bin/casewright
COPY --from=web /usr/share/nginx/html /usr/share/nginx/html
COPY deploy/live-demo/nginx.conf /etc/nginx/conf.d/default.conf
COPY deploy/live-demo/entrypoint.sh /usr/local/bin/entrypoint.sh
COPY demo/attack-scenario.log /demo/attack-scenario.log
COPY demo/watchlists /demo/watchlists
RUN chmod 0755 /usr/local/bin/entrypoint.sh && rm -f /etc/nginx/templates/*

# nginx-unprivileged keeps its pid and temp files in /tmp, so any uid works.
USER 101
EXPOSE 7860
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
