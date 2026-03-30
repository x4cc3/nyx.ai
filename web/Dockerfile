FROM node:20-alpine AS deps

ENV npm_config_fetch_retries=5 \
    npm_config_fetch_retry_factor=2 \
    npm_config_fetch_retry_maxtimeout=120000 \
    npm_config_fetch_retry_mintimeout=20000 \
    npm_config_network_timeout=300000

WORKDIR /app/web
COPY package.json package-lock.json ./
RUN for i in 1 2 3; do \
      npm ci --no-audit --no-fund && exit 0; \
      if [ "$i" -eq 3 ]; then exit 1; fi; \
      sleep $((i * 5)); \
    done

FROM node:20-alpine AS builder

WORKDIR /app/web
ARG NEXT_PUBLIC_SUPABASE_URL
ARG NEXT_PUBLIC_SUPABASE_ANON_KEY
ARG NEXT_PUBLIC_SITE_URL
ENV NEXT_TELEMETRY_DISABLED=1
ENV NEXT_PUBLIC_SUPABASE_URL=$NEXT_PUBLIC_SUPABASE_URL
ENV NEXT_PUBLIC_SUPABASE_ANON_KEY=$NEXT_PUBLIC_SUPABASE_ANON_KEY
ENV NEXT_PUBLIC_SITE_URL=$NEXT_PUBLIC_SITE_URL
COPY --from=deps /app/web/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:20-alpine AS runner

WORKDIR /app/web
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV npm_config_fetch_retries=5 \
    npm_config_fetch_retry_factor=2 \
    npm_config_fetch_retry_maxtimeout=120000 \
    npm_config_fetch_retry_mintimeout=20000 \
    npm_config_network_timeout=300000

COPY package.json package-lock.json ./
RUN for i in 1 2 3; do \
      npm ci --omit=dev --no-audit --no-fund && exit 0; \
      if [ "$i" -eq 3 ]; then exit 1; fi; \
      sleep $((i * 5)); \
    done

RUN addgroup -g 1001 -S nextjs && adduser -S nextjs -u 1001

COPY --from=builder --chown=nextjs:nextjs /app/web/.next ./.next
COPY --from=builder --chown=nextjs:nextjs /app/web/public ./public
COPY --from=builder --chown=nextjs:nextjs /app/web/next.config.js ./next.config.js

USER nextjs

EXPOSE 3000
CMD ["npm", "run", "start", "--", "--hostname", "0.0.0.0", "--port", "3000"]
