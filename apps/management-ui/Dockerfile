# Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
#
# SoftlaneIT licenses this file to you under the Apache License,
# Version 2.0 (the "LICENSE"); you may not use this file except
# in compliance with the LICENSE.

#  Stage 1: install dependencies ─
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --prefer-offline

#  Stage 2: build 
FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NEXT_TELEMETRY_DISABLED=1

# Build-time public env vars (baked into JS bundle).
# Override with ARG/ENV when running inside Docker Compose.
ARG NEXT_PUBLIC_API_URL=http://localhost:8081
ARG NEXT_PUBLIC_GATEWAY_URL=http://localhost:8081
ARG NEXT_PUBLIC_AUTH_URL=http://localhost:8082
ARG NEXT_PUBLIC_TENANT_URL=http://localhost:8083
ARG NEXT_PUBLIC_BOOKING_URL=http://localhost:8084
ARG NEXT_PUBLIC_CONFIG_URL=http://localhost:8085

ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_GATEWAY_URL=$NEXT_PUBLIC_GATEWAY_URL
ENV NEXT_PUBLIC_AUTH_URL=$NEXT_PUBLIC_AUTH_URL
ENV NEXT_PUBLIC_TENANT_URL=$NEXT_PUBLIC_TENANT_URL
ENV NEXT_PUBLIC_BOOKING_URL=$NEXT_PUBLIC_BOOKING_URL
ENV NEXT_PUBLIC_CONFIG_URL=$NEXT_PUBLIC_CONFIG_URL

RUN npm run build

#  Stage 3: minimal runtime 
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs && \
    adduser  --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
CMD ["node", "server.js"]
