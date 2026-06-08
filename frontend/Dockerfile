FROM node:24-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:24-alpine
RUN addgroup -S feednest && adduser -S feednest -G feednest
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json /app/package-lock.json ./
RUN npm ci --omit=dev
# Ensure all served assets are world-readable: the runtime runs as the non-root
# `feednest` user, and adapter-node's static file serving crashes the process on
# an EACCES read (e.g. a static file added with restrictive permissions).
RUN chmod -R a+rX ./build

USER feednest
EXPOSE 3000
ENV PORT=3000

CMD ["node", "build"]
