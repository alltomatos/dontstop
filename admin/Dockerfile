# syntax=docker/dockerfile:1

FROM node:24-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:24-alpine AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:24-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
COPY --from=build /app/.next ./.next
COPY --from=build /app/package.json /app/package-lock.json ./
COPY --from=build /app/next.config.ts ./next.config.ts
RUN npm ci --omit=dev
USER node
EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 CMD ["wget", "-q", "-O", "/dev/null", "http://127.0.0.1:3000/login"]
CMD ["npm", "run", "start", "--", "-p", "3000"]
