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
COPY --from=builder /app/package.json .
RUN npm ci --omit=dev

USER feednest
EXPOSE 3000
ENV PORT=3000

CMD ["node", "build"]
